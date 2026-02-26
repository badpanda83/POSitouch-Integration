// Package agent orchestrates reading all POSitouch DBF files and updating the cache.
package agent

import (
	"log"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

// Agent periodically reads POSitouch DBF files and updates the cache.
type Agent struct {
	config   *config.RooamConfig
	cache    *cache.Cache
	interval time.Duration
	stopCh   chan struct{}
}

// NewAgent creates a new Agent.
func NewAgent(cfg *config.RooamConfig, c *cache.Cache, interval time.Duration) *Agent {
	return &Agent{
		config:   cfg,
		cache:    c,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start performs an immediate pull then repeats every interval until Stop is called.
func (a *Agent) Start() {
	go func() {
		a.pull()
		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				a.pull()
			case <-a.stopCh:
				return
			}
		}
	}()
}

// Stop signals the agent to stop after the current pull (if any) completes.
func (a *Agent) Stop() {
	close(a.stopCh)
}

// pull reads all DBF files and updates the cache.
// Missing files are logged as warnings; they do not abort the pull.
func (a *Agent) pull() {
	dbfDir := a.config.DBFDir()
	scDir := a.config.SCDir()

	data := &cache.CachedData{
		LastUpdated: time.Now().UTC(),
	}

	// Cost centers
	cc, err := positouch.ReadCostCenters(dbfDir)
	if err != nil {
		log.Printf("[WARN] cost centers: %v", err)
	} else {
		data.CostCenters = cc
		log.Printf("[INFO] cost centers: %d records", len(cc))
	}

	// Tenders
	td, err := positouch.ReadTenders(dbfDir)
	if err != nil {
		log.Printf("[WARN] tenders: %v", err)
	} else {
		data.Tenders = td
		log.Printf("[INFO] tenders: %d records", len(td))
	}

	// Employees
	em, err := positouch.ReadEmployees(dbfDir, scDir)
	if err != nil {
		log.Printf("[WARN] employees: %v", err)
	} else {
		data.Employees = em
		log.Printf("[INFO] employees: %d records", len(em))
	}

	// Tables
	tb, err := positouch.ReadTables(dbfDir)
	if err != nil {
		log.Printf("[WARN] tables: %v", err)
	} else {
		data.Tables = tb
		log.Printf("[INFO] tables: %d records", len(tb))
	}

	// Order types
	ot, err := positouch.ReadOrderTypes(scDir)
	if err != nil {
		log.Printf("[WARN] order types: %v", err)
	} else {
		data.OrderTypes = ot
		log.Printf("[INFO] order types: %d records", len(ot))
	}

	if err := a.cache.Update(data); err != nil {
		log.Printf("[ERROR] cache update: %v", err)
		return
	}
	log.Printf("[INFO] pull complete at %s", data.LastUpdated.Format(time.RFC3339))
}
