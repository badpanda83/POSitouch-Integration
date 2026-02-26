// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and exposes it via REST endpoints.
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

// In-memory store for demonstration; production should use persistent storage.
var store = struct {
	data map[string]cache.Data
}{
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

	// ---- REST API Handlers ----

	// PUT/POST: Agent uploads cache for a location.
	http.HandleFunc("/api/v1/pos-data", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut, http.MethodPost:
			location := r.Header.Get("X-Location-ID")
			if location == "" {
				http.Error(w, "Missing X-Location-ID header", http.StatusBadRequest)
				return
			}
			var data cache.Data
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Bad data", http.StatusBadRequest)
				return
			}
			store.data[location] = data
			log.Printf("[server] received data for location %q — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d",
				location, len(data.CostCenters), len(data.Tenders), len(data.Employees), len(data.Tables), len(data.OrderTypes))
			w.WriteHeader(http.StatusOK)
			return

		case http.MethodGet:
			// List all locations and received_at timestamps.
			locations := make([]map[string]interface{}, 0, len(store.data))
			for id := range store.data {
				locations = append(locations, map[string]interface{}{
					"location":    id,
					"received_at": time.Now().UTC(), // Demo: use real timestamp if needed
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(locations)
			return

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	// GET: Retrieve all/cache entities for a location or a specific entity.
	http.HandleFunc("/api/v1/pos-data/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/pos-data/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 1 && parts[0] != "" {
			// GET /api/v1/pos-data/{location}
			locationID := parts[0]
			d, ok := store.data[locationID]
			if !ok {
				http.Error(w, "location not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(d)
			return
		}
		if len(parts) == 2 {
			locationID := parts[0]
			entity := parts[1]
			handleGetEntity(w, r, locationID, entity)
			return
		}
		http.Error(w, "Bad path", http.StatusBadRequest)
	})

	// Health check endpoint.
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

// Handler for entity endpoints: GET /api/v1/pos-data/{location}/{entity}
func handleGetEntity(w http.ResponseWriter, r *http.Request, locationID, entity string) {
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