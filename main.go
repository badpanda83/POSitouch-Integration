// POSitouch Integration Agent â€” reads POSitouch DBF files every 30 minutes,
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

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	appName   = "rooam-pos-agent"
	appVersion = "1.0.0"
	exportDir  = "C:\\Users\\Omnivore\\Documents\\POSitouch-Integration\\utils\\Export"
	tablesXML    = `C:\SC\set1.xml`
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
	fmt.Printf("â•”â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•—\n")
	fmt.Printf("â•‘  %-38sâ•‘\n", fmt.Sprintf("%s v%s", appName, appVersion))
	fmt.Printf("â•‘  %-38sâ•‘\n", "POSitouch Integration Agent")
	fmt.Printf("â•šâ•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•â•\n\n")

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

	locationID := cfg.Location.Name
	// FIX: trim any trailing slash from the configured endpoint to prevent
	// double-slash URLs like ".../pos-data//store1/categories" â†’ 400 Bad Request
	apiBaseURL := strings.TrimRight(cfg.Cloud.Endpoint, "/")
	apiKey := cfg.Cloud.APIKey

	// --- AUTOMATIC CACHE & UPLOAD FUNCTION ---
	cacheAndUpload := func() {
		log.Printf("[sync] Refreshing & uploading POSitouch entities for location: %s", locationID)

		costCenters, _ := positouch.ReadCostCenters(cfg.DBFDir)
		tenders, _ := positouch.ReadTenders(cfg.DBFDir)
		employees, _ := positouch.ReadEmployees(cfg.DBFDir, cfg.SCDir)

		// ------------- TABLES (FROM XML) -------------
		tables, err := positouch.ParseTablesFromSet1XML(tablesXML)
		if err != nil {
			log.Printf("[sync][WARN] Unable to load tables: %v", err)
			tables = nil
		} else {
			log.Printf("[sync] Loaded %d tables from %s", len(tables), tablesXML)
		}

		orderTypes, _ := positouch.ReadOrderTypes(cfg.DBFDir)
		tickets, _ := positouch.ReadAllTickets(cfg.XMLDir, cfg.XMLCloseDir)

		// ----------- MENU ITEMS -----------
		menuXMLPath := exportPath("menu_items.xml")
		menuItems, _ := positouch.ParseMenuXML(menuXMLPath)
		if len(menuItems) == 0 {
			log.Printf("[sync][WARN] No menu items loaded from %s", menuXMLPath)
		} else {
			log.Printf("[sync] Loaded %d menu items from %s", len(menuItems), menuXMLPath)
		}

		// ----------- CATEGORIES -----------
		catXMLPath := exportPath("menu_categories.xml")
		categories, _ := positouch.ParseMenuCategories(catXMLPath)
		if len(categories) == 0 {
			log.Printf("[sync][WARN] No categories loaded from %s", catXMLPath)
		} else {
			log.Printf("[sync] Loaded %d categories from %s", len(categories), catXMLPath)
		}

		// ----------- MODIFIERS -----------
		modXMLPath := exportPath("menu_items.xml")
		modifiers, _ := positouch.ParseMenuModifiers(modXMLPath)
		if len(modifiers) == 0 {
			log.Printf("[sync][WARN] No modifiers loaded from %s", modXMLPath)
		} else {
			log.Printf("[sync] Loaded %d modifiers from %s", len(modifiers), modXMLPath)
		}

		d := cache.Data{
			CostCenters:       costCenters,
			Tenders:           tenders,
			Employees:         employees,
			Tables:            tables,
			OrderTypes:        orderTypes,
			CurrentTickets:    tickets,
			HistoricalTickets: nil,
			MenuItems:         menuItems,
			Categories:        categories,
			Modifiers:         modifiers,
			LastUpdated:       time.Now(),
		}
		c.Update(d)

		cached := c.Get()
		entityMap := map[string]interface{}{
			"employees":    cached.Employees,
			"tables":       cached.Tables,
			"tenders":      cached.Tenders,
			"cost_centers": cached.CostCenters,
			"order_types":  cached.OrderTypes,
			"tickets":      cached.CurrentTickets,
			"menu_items":   cached.MenuItems,
			"categories":   cached.Categories,
			"modifiers":    cached.Modifiers,
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
			req.Header.Set("Authorization", "Bearer "+apiKey)

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
			log.Printf("[server] received data for location %q â€” cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d tickets=%d menu_items=%d categories=%d modifiers=%d",
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
			return
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	http.HandleFunc("/api/v1/pos-data/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/pos-data/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 1 && parts[0] != "" {
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

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	go func() {
		log.Println("[server] starting API on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Printf("[main] received signal %s â€” shutting down", sig)
	log.Println("[main] Agent stopped")
}

func exportPath(filename string) string {
	return fmt.Sprintf("%s\\%s", exportDir, filename)
}

func countItems(arr interface{}) int {
	switch v := arr.(type) {
	case []interface{}:
		return len(v)
	case []positouch.Employee:
		return len(v)
	case []positouch.Table:
		return len(v)
	case []positouch.Tender:
		return len(v)
	case []positouch.CostCenter:
		return len(v)
	case []positouch.OrderType:
		return len(v)
	case []positouch.Ticket:
		return len(v)
	case []positouch.MenuItem:
		return len(v)
	case []positouch.Category:
		return len(v)
	case []positouch.Modifier:
		return len(v)
	default:
		return -1
	}
}

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
	case "tickets":
		json.NewEncoder(w).Encode(d.CurrentTickets)
	case "menu_items":
		json.NewEncoder(w).Encode(d.MenuItems)
	case "categories":
		json.NewEncoder(w).Encode(d.Categories)
	case "modifiers":
		json.NewEncoder(w).Encode(d.Modifiers)
	default:
		http.Error(w, "entity not found", http.StatusNotFound)
	}
}

