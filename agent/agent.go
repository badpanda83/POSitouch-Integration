// Package agent implements the main pull loop for the POSitouch integration.
package agent

import (
	"log"
	"path/filepath"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const pullInterval = 30 * time.Minute

// Agent orchestrates the periodic reading of POSitouch DBF files and the
// updating of the local JSON cache.
type Agent struct {
	cfg       *config.AppConfig
	cache     *cache.Cache
	cachePath string
	ticker    *time.Ticker
	done      chan struct{}
}

// New creates an Agent using the supplied config.  The cache file is written
// alongside the config file (same directory).
func New(cfg *config.AppConfig, configPath string) *Agent {
	cacheDir := filepath.Dir(configPath)
	return &Agent{
		cfg:       cfg,
		cache:     cache.New(),
		cachePath: filepath.Join(cacheDir, "rooam_cache.json"),
		done:      make(chan struct{}),
	}
}

// Start loads any existing cache, performs an immediate data pull, then
// starts a 30-minute ticker that repeats the pull on each interval.
// Start returns immediately; the pull loop runs in a background goroutine.
func (a *Agent) Start() {
	if err := a.cache.Load(a.cachePath); err != nil {
		log.Printf("agent: warning: could not load existing cache: %v", err)
	} else {
		log.Printf("agent: loaded existing cache from %s", a.cachePath)
	}

	// Perform an immediate pull before waiting for the first tick.
	a.pullAll()

	a.ticker = time.NewTicker(pullInterval)
	go func() {
		for {
			select {
			case <-a.ticker.C:
				a.pullAll()
			case <-a.done:
				return
			}
		}
	}()
	log.Printf("agent: started — pulling every %s", pullInterval)
}

// Stop signals the background goroutine to exit and waits for it to finish.
func (a *Agent) Stop() {
	if a.ticker != nil {
		a.ticker.Stop()
	}
	close(a.done)
	log.Println("agent: stopped")
}

// pullAll reads all five POSitouch data categories, updates the in-memory
// cache, and persists it to disk.  Individual read failures are logged but do
// not prevent the remaining reads from completing.
func (a *Agent) pullAll() {
	start := time.Now()
	log.Println("agent: starting data pull")

	dbfDir := a.cfg.DBFPath()
	scDir := a.cfg.SCPath()

	costCenters, err := positouch.ReadCostCenters(dbfDir)
	if err != nil {
		log.Printf("agent: error reading cost centers: %v", err)
	}

	tenders, err := positouch.ReadTenders(dbfDir)
	if err != nil {
		log.Printf("agent: error reading tenders: %v", err)
	}

	employees, err := positouch.ReadEmployees(dbfDir, scDir)
	if err != nil {
		log.Printf("agent: error reading employees: %v", err)
	}

	tables, err := positouch.ReadTables(dbfDir)
	if err != nil {
		log.Printf("agent: error reading tables: %v", err)
	}

	orderTypes, err := positouch.ReadOrderTypes(dbfDir)
	if err != nil {
		log.Printf("agent: error reading order types: %v", err)
	}

	a.cache.Update(costCenters, tenders, employees, tables, orderTypes)

	if err := a.cache.Save(a.cachePath); err != nil {
		log.Printf("agent: error saving cache: %v", err)
	} else {
		log.Printf("agent: cache saved to %s", a.cachePath)
	}

	log.Printf("agent: pull complete in %s — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d",
		time.Since(start).Round(time.Millisecond),
		len(costCenters), len(tenders), len(employees), len(tables), len(orderTypes),
	)
}
