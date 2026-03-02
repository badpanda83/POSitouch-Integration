// Package agent implements the main refresh loop for the POSitouch integration.
// It reads all POSitouch DBF files (and XML exports) every 30 minutes and updates the cache.
package agent

import (
	"log"
	"path/filepath"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// RefreshInterval is the time between successive data pulls.
const RefreshInterval = 30 * time.Minute

// ExportDir is where the XML menu and category files are located.
const ExportDir = "C:\\Users\\Omnivore\\Documents\\POSitouch-Integration\\utils\\Export"

// Agent orchestrates periodic data pulls from POSitouch sources.
type Agent struct {
	cfg   *config.Config
	cache *cache.Cache
	stop  chan struct{}
	done  chan struct{}
}

// New creates a new Agent using the provided configuration and cache.
func New(cfg *config.Config, c *cache.Cache) *Agent {
	return &Agent{
		cfg:   cfg,
		cache: c,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Start performs an immediate data pull then schedules subsequent pulls every
// 30 minutes. It blocks until Stop is called.
func (a *Agent) Start() {
	defer close(a.done)

	log.Println("[agent] starting — performing initial data pull")
	a.refresh()

	ticker := time.NewTicker(RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("[agent] scheduled refresh triggered")
			a.refresh()
		case <-a.stop:
			log.Println("[agent] shutdown signal received — stopping")
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

	tables, err := positouch.ReadTables(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: tables: %v", err)
	}
	cache.WriteTablesToCache(emptyIfNil(tables), filepath.Join(cacheDir, "tables.cache"))

	orderTypes, err := positouch.ReadOrderTypes(scDir)
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

	// ----- MENU ITEMS (from menu_items.xml in ExportDir) -----
	menuItems := []positouch.MenuItem{}
	menuXMLPath := filepath.Join(ExportDir, "menu_items.xml")
	menuExport, err := positouch.ParseMenuXML(menuXMLPath)
	if err != nil {
		log.Printf("[agent] WARNING: ParseMenuXML (%s): %v", menuXMLPath, err)
		menuItems = []positouch.MenuItem{}
	} else {
		menuItems = menuExport
		log.Printf("[agent] Parsed %d menu items from %s", len(menuItems), menuXMLPath)
	}
	cache.WriteMenuItemsToCache(emptyIfNil(menuItems), filepath.Join(cacheDir, "menu_items.json"))

	// ----- CATEGORIES (from menu_categories.xml in ExportDir) -----
	categories := []positouch.Category{}
	catXMLPath := filepath.Join(ExportDir, "menu_categories.xml")
	cats, err := positouch.ParseMenuCategories(catXMLPath)
	if err != nil {
		log.Printf("[agent] WARNING: ParseMenuCategories (%s): %v", catXMLPath, err)
		categories = []positouch.Category{}
	} else {
		categories = cats
		log.Printf("[agent] Parsed %d categories from %s", len(categories), catXMLPath)
	}
	cache.WriteCategoriesToCache(emptyIfNil(categories), filepath.Join(cacheDir, "categories.json"))

	// ----- MODIFIERS (from menu_modifiers.xml in ExportDir, if available) -----
	modifiers := []positouch.Modifier{}
	modXMLPath := filepath.Join(ExportDir, "menu_modifiers.xml")
	if _, err := os.Stat(modXMLPath); err == nil {
		mods, err := positouch.ParseMenuModifiers(modXMLPath)
		if err != nil {
			log.Printf("[agent] WARNING: ParseMenuModifiers (%s): %v", modXMLPath, err)
		} else {
			modifiers = mods
			log.Printf("[agent] Parsed %d modifiers from %s", len(modifiers), modXMLPath)
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
		log.Printf("[agent] Cloud sync enabled — update logic as needed for file-based cache")
	}
}

func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}