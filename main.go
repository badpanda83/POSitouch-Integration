// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and persists it to rooam_cache.json.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

const (
	appName    = "rooam-pos-agent"
	appVersion = "1.0.0"
)

// Dummy store - in production, use the actual store from your server file.
// Here, we simulate cache for demonstration.
var store = struct {
	mu   chan struct{}       // Not strictly correct, for full thread safety use sync.RWMutex
	data map[string]cache.Data
}{
	mu:   make(chan struct{}, 1),
	data: make(map[string]cache.Data),
}

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	fmt.Printf("╔══════════════════════════════════════════╗\n")
	fmt.Printf("║  %-38s║\n", fmt.Sprintf("%s v%s", appName, appVersion))
	fmt.Printf("║  %-38s║\n", "POSitouch Integration Agent")
	fmt.Printf("╚══════════════════════════════════════════╝\n\n")

	log.Printf("[main] config path : %s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[main] failed to load config: %v", err)
	}

	log.Printf("[main] location    : %s", cfg.Location.Name)
	log.Printf("[main] install dir : %s", cfg.InstallDir)
	log.Printf("[main] SC dir      : %s", cfg.SCDir)
	log.Printf("[main] DBF dir     : %s", cfg.DBFDir)
	log.Printf("[main] ALTDBF dir  : %s", cfg.AltDBFDir)

	c := cache.New(cfg.InstallDir)
	a := agent.New(cfg, c)

	// Set up HTTP endpoints to serve REST data
	http.HandleFunc("/api/v1/pos-data/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/pos-data/")
		if path == "" || strings.Contains(path, "/") == false {
			// Show all locations or a specific location as before
			handleGetLocation(w, r)
			return
		}

		parts := strings.SplitN(path, "/", 2)
		locationID := parts[0]
		entity := ""
		if len(parts) == 2 {
			entity = parts[1]
		}

		handleGetEntity(w, r, locationID, entity)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	go func() {
		log.Println("[server] starting API on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Agent run/shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("[main] received signal %s — shutting down", sig)
		a.Stop()
	}()

	a.Start()
	log.Println("[main] Agent stopped")
}

// Handler for base entity endpoints e.g. /api/v1/pos-data/My%20Test%20Restaurant/employees
func handleGetEntity(w http.ResponseWriter, r *http.Request, locationID, entity string) {
	// Example: store.data is map[string]cache.Data, adjust if you use another struct
	d, ok := store.data[locationID]
	if !ok {
		http.Error(w, "location not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	switch entity {
	case "tables":
		json.NewEncoder(w).Encode(d.Tables)
	case "employees":
		json.NewEncoder(w).Encode(d.Employees)
	case "tenders":
		json.NewEncoder(w).Encode(d.Tenders)
	case "cost_centers":
		json.NewEncoder(w).Encode(d.CostCenters)
	case "order_types":
		json.NewEncoder(w).Encode(d.OrderTypes)
	default:
		http.Error(w, "entity not found", http.StatusNotFound)
	}
}

// Handler for /api/v1/pos-data or /api/v1/pos-data/{location}
func handleGetLocation(w http.ResponseWriter, r *http.Request) {
	// If path is empty, show all available locations and timestamps
	locationID := strings.TrimPrefix(r.URL.Path, "/api/v1/pos-data/")
	if locationID == "" {
		locations := make([]map[string]interface{}, 0, len(store.data))
		for id, loc := range store.data {
			locations = append(locations, map[string]interface{}{
				"location":    id,
				"received_at": time.Now().UTC(), // In real server, use your timestamp
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(locations)
		return
	}

	d, ok := store.data[locationID]
	if !ok {
		http.Error(w, "location not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}