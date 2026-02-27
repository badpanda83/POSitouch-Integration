// Package agent implements the main refresh loop for the POSitouch integration.
// It reads all POSitouch DBF files every 30 minutes and updates the cache.
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

// Agent orchestrates periodic data pulls from POSitouch DBF files.
type Agent struct {
	cfg   *config.Config
	cache *cache.Cache // Not used for file separation, but can still be used if needed
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

// refresh reads all POSitouch DBF files and updates all caches in data_cache/
func (a *Agent) refresh() {
	dbfDir := a.cfg.DBFDir
	scDir := a.cfg.SCDir
	xmlOpenDir := a.cfg.XMLDir                      // OMNIVORE_OPEN
	xmlCloseDir := a.cfg.XMLCloseDir                // OMNIVORE_CLOSE

	cacheDir := filepath.Join(a.cfg.InstallDir, "data_cache")
	log.Printf("[agent] reading DBF files from %s", dbfDir)
	log.Printf("[agent] reading XML ticket files from %s and %s", xmlOpenDir, xmlCloseDir)
	log.Printf("[agent] writing cache files to %s", cacheDir)

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

	allTickets, ticketErr := positouch.ReadAllTickets(xmlOpenDir, xmlCloseDir)
	if ticketErr != nil {
		log.Printf("[agent] WARNING: tickets: %v", ticketErr)
	}
	// Combine open and closed tickets together as one slice for tickets.cache
	cache.WriteTicketsToCache(emptyIfNil(allTickets), filepath.Join(cacheDir, "tickets.cache"))

	log.Printf("[agent] refreshed and wrote each cache in data_cache/")

	// ----------- CLOUD SYNC ADDITION STARTS HERE -----------
	// Use cache files for cloud sync, or keep using the combined struct if that's required
	// You can reconstruct d from files if needed, or use the struct as before
	// This example just logs that file-based caches are written
	if a.cfg.Cloud.Enabled {
		log.Printf("[agent] Cloud sync enabled — update logic as needed for file-based cache")
	}
	// ----------- CLOUD SYNC ADDITION ENDS HERE -----------
}

func emptyIfNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}