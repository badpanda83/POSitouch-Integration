// Package agent implements the main refresh loop for the POSitouch integration.
// It reads all POSitouch DBF files (and XML exports) every 30 minutes and updates the cache.
package agent

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// RefreshInterval is the time between successive data pulls.
const RefreshInterval = 30 * time.Minute

// --- Robust WExport runner with logging ---

var wexportMu sync.Mutex

func runWExportAndCopyForTables(exportDir string) error {
	wexportMu.Lock()
	defer wexportMu.Unlock()

	// 1. Generate fresh set1.xml using WExport.exe, capturing output for logs
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		"C:\\SC\\WExport.EXE",
		"ExportSettings",
		"C:\\Users\\Omnivore\\Documents\\POSitouch-Integration\\utils\\wexport_layout_manifest.xml",
	)
	cmd.Dir = positouch.WExportDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("[WExport] Running export command: %v", cmd.Args)
	err := cmd.Run()
	log.Printf("[WExport] STDOUT:\n%s", stdout.String())
	if stderr.Len() > 0 {
		log.Printf("[WExport] STDERR:\n%s", stderr.String())
	}
	if err != nil {
		log.Printf("[WExport][ERROR] Exit status: %v", err)
		return err
	} else {
		log.Printf("[WExport][SUCCESS] Export completed successfully.")
	}

	// 2. Copy set1.xml to Export folder
	src := positouch.Set1XMLSrc
	dst := filepath.Join(exportDir, "set1.xml")
	in, err := os.Open(src)
	if err != nil {
		log.Printf("[WExport][ERROR] reading %s: %v", src, err)
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		log.Printf("[WExport][ERROR] creating %s: %v", dst, err)
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		log.Printf("[WExport][ERROR] copying set1.xml: %v", err)
		return err
	}
	log.Printf("[WExport] set1.xml copied successfully to %s", dst)
	return nil
}

// Agent orchestrates periodic data pulls from POSitouch sources.
type Agent struct {
	cfg   *config.Config
	cache *cache.Cache
	stop  chan struct{}
	done  chan struct{}
}

// ... rest unchanged ...

// New creates a new Agent using the provided configuration and cache.
func New(cfg *config.Config, c *cache.Cache) *Agent {
	return &Agent{
		cfg:   cfg,
		cache: c,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Start performs an immediate data pull then schedules subsequent pulls every 30 minutes. It blocks until Stop is called.
func (a *Agent) Start() {
	defer close(a.done)

	log.Println("[agent] starting â€” performing initial data pull")
	a.refresh()

	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("[agent] scheduled refresh triggered")
			a.refresh()
		case <-a.stop:
			log.Println("[agent] shutdown signal received â€” stopping")
			return
		}
	}
}

// Stop signals the agent to stop and waits for the current operation to finish.
func (a *Agent) Stop() {
	close(a.stop)
	<-a.done
}

// refresh reads all POSitouch files and updates all caches in data_cache/
func (a *Agent) refresh() {
	dbfDir := a.cfg.DBFDir
	scDir := a.cfg.SCDir
	xmlOpenDir := a.cfg.XMLDir
	xmlCloseDir := a.cfg.XMLCloseDir

	exportDir := filepath.Join(a.cfg.InstallDir, "Export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Printf("[agent] WARNING: could not create Export dir %s: %v", exportDir, err)
	}

	cacheDir := filepath.Join(a.cfg.InstallDir, "data_cache")
	log.Printf("[agent] reading DBF files from %s", dbfDir)
	log.Printf("[agent] reading XML ticket files from %s and %s", xmlOpenDir, xmlCloseDir)
	log.Printf("[agent] writing cache files to %s", cacheDir)

	// ----- Standard DBF-based entities -----
	costCenters, err := positouch.ReadCostCenters(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: cost centers: %v", err)
	}
	cache.WriteCostCentersToCache(emptyIfNil(costCenters), filepath.Join(cacheDir, "cost_centers.cache"))

	tenders, err := positouch.ReadTenders(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: tenders: %v", err)
	}
	cache.WriteTendersToCache(emptyIfNil(tenders), filepath.Join(cacheDir, "tenders.cache"))

	employees, err := positouch.ReadEmployees(dbfDir, scDir)
	if err != nil {
		log.Printf("[agent] WARNING: employees: %v", err)
	}
	cache.WriteEmployeesToCache(emptyIfNil(employees), filepath.Join(cacheDir, "employees.cache"))

	// --- Improved: WExport runner logs every detail ---
	if err := runWExportAndCopyForTables(exportDir); err != nil {
		log.Printf("[agent] ERROR: failed to run WExport/copy set1.xml: %v", err)
	} else {
		log.Printf("[agent] set1.xml was regenerated and copied to Export dir successfully.")
	}
	tables, err := positouch.ParseTablesFromSet1XML(filepath.Join(exportDir, "set1.xml"))
	if err != nil {
		log.Printf("[sync][WARN] Unable to load tables: %v", err)
		tables = nil
	}
	cache.WriteTablesToCache(emptyIfNil(tables), filepath.Join(cacheDir, "tables.cache"))

	orderTypes, err := positouch.ReadOrderTypes(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: order types: %v", err)
	}
	cache.WriteOrderTypesToCache(emptyIfNil(orderTypes), filepath.Join(cacheDir, "order_types.cache"))

	// ----- Tickets -----
	allTickets, ticketErr := positouch.ReadAllTickets(xmlOpenDir, xmlCloseDir)
	if ticketErr != nil {
		log.Printf("[agent] WARNING: tickets: %v", ticketErr)
	}
	cache.WriteTicketsToCache(emptyIfNil(allTickets), filepath.Join(cacheDir, "tickets.cache"))
	log.Printf("[agent] OMNIVORE_OPEN: cached %d tickets", len(allTickets))

	// ----- MENU ITEMS (from menu_items.xml in exportDir) -----
	menuItems := []positouch.MenuItem{}
	menuXMLPath := filepath.Join(exportDir, "menu_items.xml")
	log.Printf("[agent] Attempting to load menu items from: %s", menuXMLPath)
	menuExport, err := positouch.ParseMenuXML(menuXMLPath)
	if err != nil {
		log.Printf("[agent] WARNING: ParseMenuXML (%s): %v", menuXMLPath, err)
		menuItems = []positouch.MenuItem{}
	} else {
		menuItems = menuExport
		log.Printf("[agent] Parsed %d menu items from %s", len(menuItems), menuXMLPath)
		for i, item := range menuItems {
			if i < 5 {
				log.Printf("[agent] MenuItem[%d]: %+v", i, item)
			}
		}
	}
	cache.WriteMenuItemsToCache(emptyIfNil(menuItems), filepath.Join(cacheDir, "menu_items.json"))

	// ----- CATEGORIES (from menu_categories.xml in exportDir) -----
	categories := []positouch.Category{}
	catXMLPath := filepath.Join(exportDir, "menu_categories.xml")
	log.Printf("[agent] [DEBUG] Attempting to load categories from: %s", catXMLPath)
	cats, err := positouch.ParseMenuCategories(catXMLPath)
	if err != nil {
		log.Printf("[agent] WARNING: ParseMenuCategories (%s): %v", catXMLPath, err)
		categories = []positouch.Category{}
	} else {
		categories = cats
		log.Printf("[agent] [DEBUG] Parsed %d categories from %s", len(categories), catXMLPath)
		for i, category := range categories {
			if i < 10 { // Only print first 10 for brevity
				log.Printf("[agent] [DEBUG] Category[%d]: %+v", i, category)
			}
		}
		if len(categories) > 10 {
			log.Printf("[agent] [DEBUG] ...and %d more categories.", len(categories)-10)
		}
	}
	cache.WriteCategoriesToCache(emptyIfNil(categories), filepath.Join(cacheDir, "categories.json"))

	// ----- MODIFIERS (from menu_modifiers.xml in exportDir, if available) -----
	modifiers := []positouch.Modifier{}
	modXMLPath := filepath.Join(exportDir, "menu_modifiers.xml")
	if _, err := os.Stat(modXMLPath); err == nil {
		log.Printf("[agent] Attempting to load modifiers from: %s", modXMLPath)
		mods, err := positouch.ParseMenuModifiers(modXMLPath)
		if err != nil {
			log.Printf("[agent] WARNING: ParseMenuModifiers (%s): %v", modXMLPath, err)
		} else {
			modifiers = mods
			log.Printf("[agent] Parsed %d modifiers from %s", len(modifiers), modXMLPath)
			for i, mod := range modifiers {
				if i < 5 {
					log.Printf("[agent] Modifier[%d]: %+v", i, mod)
				}
			}
		}
	}
	cache.WriteModifiersToCache(emptyIfNil(modifiers), filepath.Join(cacheDir, "modifiers.json"))

	// ----- Combined cache data object -----
	allData := cache.Data{
		LastUpdated:        time.Now().UTC(),
		CostCenters:        emptyIfNil(costCenters),
		Tenders:            emptyIfNil(tenders),
		Employees:          emptyIfNil(employees),
		Tables:             emptyIfNil(tables),
		OrderTypes:         emptyIfNil(orderTypes),
		CurrentTickets:     emptyIfNil(allTickets),
		HistoricalTickets:  []positouch.Ticket{},
		MenuItems:          emptyIfNil(menuItems),
		Modifiers:          emptyIfNil(modifiers),
		Categories:         emptyIfNil(categories),
	}
	a.cache.Update(allData)

	log.Printf("[agent] refreshed and wrote each cache in data_cache/")

	// ----- CLOUD SYNC OPTIONAL HOOK -----
	if a.cfg.Cloud.Enabled {
		log.Printf("[agent] Cloud sync enabled â€” update logic as needed for file-based cache")
	}
}

func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}