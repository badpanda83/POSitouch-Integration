// Package agent implements the main polling loop that reads POSitouch DBF files
// every 30 minutes and stores the results in the cache.
package agent

import (
	"log"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const refreshInterval = 30 * time.Minute

// Agent polls the POSitouch DBF files on a regular interval.
type Agent struct {
	cfg   *config.Config
	cache *cache.Cache
	done  chan struct{}
}

// New creates a new Agent using the given config and cache.
func New(cfg *config.Config, c *cache.Cache) *Agent {
	return &Agent{
		cfg:   cfg,
		cache: c,
		done:  make(chan struct{}),
	}
}

// Start kicks off the initial refresh and then schedules subsequent refreshes
// every 30 minutes. It is non-blocking — it runs in a separate goroutine.
func (a *Agent) Start() {
	go a.run()
}

// Stop signals the agent to stop after its current cycle (if any).
func (a *Agent) Stop() {
	close(a.done)
}

func (a *Agent) run() {
	log.Println("agent: starting — performing initial data pull")
	a.refresh()

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("agent: scheduled refresh triggered")
			a.refresh()
		case <-a.done:
			log.Println("agent: stopping")
			return
		}
	}
}

// refresh reads all DBF files and updates the cache.
// Missing files are logged as warnings rather than fatal errors.
func (a *Agent) refresh() {
	log.Println("agent: refresh started")
	start := time.Now()

	dbfDir := a.cfg.DBFDir()
	altDbfDir := a.cfg.AltDBFDir()
	scDir := a.cfg.SCDir()

	d := cache.Data{}

	// Cost Centers
	cc, err := positouch.ReadCostCenters(dbfDir, altDbfDir)
	if err != nil {
		log.Printf("agent: warning — cost centers: %v", err)
	} else {
		d.CostCenters = cc
	}

	// Tenders
	tenders, err := positouch.ReadTenders(dbfDir, altDbfDir)
	if err != nil {
		log.Printf("agent: warning — tenders: %v", err)
	} else {
		d.Tenders = tenders
	}

	// Employees
	employees, err := positouch.ReadEmployees(dbfDir, altDbfDir, scDir)
	if err != nil {
		log.Printf("agent: warning — employees: %v", err)
	} else {
		d.Employees = employees
	}

	// Tables
	tables, err := positouch.ReadTables(dbfDir, altDbfDir)
	if err != nil {
		log.Printf("agent: warning — tables: %v", err)
	} else {
		d.Tables = tables
	}

	// Order Types (from SC dir, not DBF dir)
	orderTypes, err := positouch.ReadOrderTypes(scDir)
	if err != nil {
		log.Printf("agent: warning — order types: %v", err)
	} else {
		d.OrderTypes = orderTypes
	}

	if err := a.cache.Update(d); err != nil {
		log.Printf("agent: error — updating cache: %v", err)
	} else {
		log.Printf("agent: refresh complete in %v (cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d)",
			time.Since(start),
			len(d.CostCenters), len(d.Tenders), len(d.Employees), len(d.Tables), len(d.OrderTypes))
	}
}
