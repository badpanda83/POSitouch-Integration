// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and exposes it via REST endpoints.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/badpanda83/POSitouch-Integration/auth"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/driver"
	micros3700driver "github.com/badpanda83/POSitouch-Integration/driver/micros3700"
	positouchdriver "github.com/badpanda83/POSitouch-Integration/driver/positouch"
	"github.com/badpanda83/POSitouch-Integration/entities"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	appName    = "rooam-pos-agent"
	appVersion = "1.0.0"
)

var store = struct {
	data map[string]cache.Data
}{
	data: make(map[string]cache.Data),
}

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	fmt.Printf("╔═══════════════════════════════════════════╗\n")
	fmt.Printf("║  %-38s║\n", fmt.Sprintf("%s v%s", appName, appVersion))
	fmt.Printf("║  %-38s║\n", "POSitouch Integration Agent")
	fmt.Printf("╚═══════════════════════════════════════════╝\n\n")

	// If started by the Windows SCM, runAsWindowsService starts the SCM
	// dispatch loop in a goroutine and returns (true, stopCh).
	// We block on stopCh until the SCM sends Stop/Shutdown, then exit cleanly.
	isSvc, stop := runAsWindowsService("RooamPOSAgent")
	if isSvc {
		<-stop
		log.Println("[main] SCM stop received — shutting down")
		os.Exit(0)
	}

	log.Printf("[main] config path : %s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[main] failed to load config: %v", err)
	}

	var posDriver driver.POSDriver
	switch cfg.EffectivePOSType() {
	case "positouch":
		posDriver = positouchdriver.New(cfg)
	case "micros3700":
		posDriver = micros3700driver.New(cfg)
	default:
		log.Fatalf("[main] unknown pos_type: %q", cfg.EffectivePOSType())
	}
	log.Printf("[main] POS driver     : %s", posDriver.Name())

	log.Printf("[main] location    : %s", cfg.Location.Name)
	log.Printf("[main] install dir : %s", cfg.InstallDir)
	log.Printf("[main] SC dir      : %s", cfg.SCDir)
	log.Printf("[main] DBF dir     : %s", cfg.DBFDir)
	log.Printf("[main] ALTDBF dir  : %s", cfg.AltDBFDir)

	exportDir := filepath.Join(cfg.InstallDir, "Export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Printf("[main] WARNING: could not create Export dir %s: %v", exportDir, err)
	}
	log.Printf("[main] export dir  : %s", exportDir)

	locationID := cfg.Location.Name
	// FIX: trim any trailing slash from the configured endpoint to prevent
	// double-slash URLs like ".../pos-data//store1/categories" → 400 Bad Request
	apiBaseURL := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	// TODO(phase-3b): when AuthMode == "oauth", construct auth.OAuthProvider instead.
	var tokenProvider auth.TokenProvider = &auth.StaticKeyProvider{Key: cfg.Cloud.APIKey}

	// Kill any stale WExport.EXE processes left from previous runs.
	// taskkill returns a non-zero exit code when no matching process exists,
	// which is the normal/expected case — only log when we actually killed one.
	if err := exec.Command("taskkill", "/F", "/IM", "WExport.EXE").Run(); err == nil {
		log.Printf("[main] killed stale WExport.EXE process(es)")
	}

	// --- AUTOMATIC CACHE & UPLOAD FUNCTION ---
	cacheAndUpload := func() {
		log.Printf("[sync] Refreshing & uploading POSitouch entities for location: %s", locationID)

		snapshot, syncErr := posDriver.SyncEntities()
		if syncErr != nil {
			log.Printf("[sync][ERROR] SyncEntities failed: %v", syncErr)
			return
		}

		tickets, tickErr := posDriver.SyncTickets()
		if tickErr != nil {
			log.Printf("[sync][WARN] SyncTickets failed: %v", tickErr)
		}

		entityMap := map[string]interface{}{
			"employees":    snapshot.Employees,
			"tables":       snapshot.Tables,
			"tenders":      snapshot.Tenders,
			"cost_centers": snapshot.CostCenters,
			"order_types":  snapshot.OrderTypes,
			"tickets":      tickets,
			"menu_items":   snapshot.MenuItems,
		}

		for entity, arr := range entityMap {
			l := countItems(arr)
			log.Printf("[sync] preparing to upload %d %s", l, entity)
			if l == 0 {
				log.Printf("[sync][WARN] Entity %s is EMPTY", entity)
			}
		}

		for entity, arr := range entityMap {
			data, err := json.Marshal(arr)
			if err != nil {
				log.Printf("[sync] failed to marshal %s: %v", entity, err)
				continue
			}
			// apiBaseURL already has trailing slash stripped above, so this
			// always produces: <endpoint>/<locationID>/<entity>  (no double slash)
			url := fmt.Sprintf("%s/%s/%s", apiBaseURL, locationID, entity)
			req, err := http.NewRequest("PUT", url, strings.NewReader(string(data)))
			if err != nil {
				log.Printf("[sync] failed to create request for %s: %v", entity, err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			token, err := tokenProvider.GetAccessToken()
			if err != nil {
				log.Printf("[sync] failed to get access token for %s: %v", entity, err)
				continue
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("[sync] failed to upload %s: %v", entity, err)
				continue
			}
			log.Printf("[sync] uploaded %s, response status: %s", entity, resp.Status)
			resp.Body.Close()
		}
		log.Printf("[sync] All entities uploaded for location: %s", locationID)
	}
	cacheAndUpload()

	go func() {
		for {
			time.Sleep(30 * time.Minute)
			cacheAndUpload()
		}
	}()

	// --- FAST TICKET REFRESH FUNCTION ---
	refreshTickets := func() {
		tickets, err := posDriver.SyncTickets()
		if err != nil {
			log.Printf("[ticket_sync] error reading tickets: %v", err)
			return
		}
		log.Printf("[ticket_sync] found %d tickets", len(tickets))

		data, err := json.Marshal(tickets)
		if err != nil {
			log.Printf("[ticket_sync] failed to marshal tickets: %v", err)
			return
		}
		url := fmt.Sprintf("%s/%s/tickets", apiBaseURL, locationID)
		req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
		if err != nil {
			log.Printf("[ticket_sync] failed to create request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		token, err := tokenProvider.GetAccessToken()
		if err != nil {
			log.Printf("[ticket_sync] failed to get access token: %v", err)
			return
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[ticket_sync] failed to upload tickets: %v", err)
			return
		}
		resp.Body.Close()
		log.Printf("[ticket_sync] uploaded tickets, response status: %s", resp.Status)
	}

	go func() {
		for {
			time.Sleep(30 * time.Second)
			refreshTickets()
		}
	}()
	log.Println("[ticket_sync] polling tickets every 30s")

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
			log.Printf("[server] received data for location %q — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d tickets=%d menu_items=%d categories=%d modifiers=%d",
				location, len(data.CostCenters), len(data.Tenders), len(data.Employees), len(data.Tables), len(data.OrderTypes), len(data.CurrentTickets), len(data.MenuItems), len(data.Categories), len(data.Modifiers))
			w.WriteHeader(http.StatusOK)
			return
		case http.MethodGet:
			locations := make([]map[string]interface{}, 0, len(store.data))
			for id, d := range store.data {
				locations = append(locations, map[string]interface{}{
					"location":    id,
					"received_at": time.Now().UTC(),
					"summary": map[string]int{
						"cost_centers": len(d.CostCenters),
						"tenders":      len(d.Tenders),
						"employees":    len(d.Employees),
						"tables":       len(d.Tables),
						"order_types":  len(d.OrderTypes),
						"tickets":      len(d.CurrentTickets),
						"menu_items":   len(d.MenuItems),
						"categories":   len(d.Categories),
						"modifiers":    len(d.Modifiers),
					},
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(locations)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// REST endpoints for individual entity types
	entityEndpoints := map[string]func() (interface{}, error){
		"/api/v1/employees":    func() (interface{}, error) { return posDriver.SyncEntities() },
		"/api/v1/tickets":      func() (interface{}, error) { return posDriver.SyncTickets() },
		"/api/v1/menu-items":   func() (interface{}, error) { s, e := posDriver.SyncEntities(); return s.MenuItems, e },
		"/api/v1/tables":       func() (interface{}, error) { s, e := posDriver.SyncEntities(); return s.Tables, e },
		"/api/v1/tenders":      func() (interface{}, error) { s, e := posDriver.SyncEntities(); return s.Tenders, e },
		"/api/v1/cost-centers": func() (interface{}, error) { s, e := posDriver.SyncEntities(); return s.CostCenters, e },
		"/api/v1/order-types":  func() (interface{}, error) { s, e := posDriver.SyncEntities(); return s.OrderTypes, e },
	}
	for path, fn := range entityEndpoints {
		path, fn := path, fn
		http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			result, err := fn()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	go func() {
		port := ":8080"
		log.Printf("[server] listening on %s", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatalf("[server] ListenAndServe: %v", err)
		}
	}()

	// --- GRACEFUL SHUTDOWN (interactive mode only) ---
	// In service mode the <-stop block above owns the lifecycle and returns
	// before we get here.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	log.Println("[main] received OS signal — shutting down")
}

// countItems returns the length of a slice passed as interface{}. 
func countItems(v interface{}) int {
	switch s := v.(type) {
	case []entities.Employee:
		return len(s)
	case []entities.Table:
		return len(s)
	case []entities.Tender:
		return len(s)
	case []entities.CostCenter:
		return len(s)
	case []entities.OrderType:
		return len(s)
	case []entities.MenuItem:
		return len(s)
	case []positouch.Ticket:
		return len(s)
	}
	return 0
}