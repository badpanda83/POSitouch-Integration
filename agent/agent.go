// Package agent implements the main refresh loop for the POSitouch integration.
// It reads all POSitouch DBF files every 30 minutes and updates the cache.
package agent

import (
	"log"
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

// refresh reads all POSitouch DBF files and updates the cache.
// Errors from individual file reads are logged but do not abort the refresh.
func (a *Agent) refresh() {
	dbfDir := a.cfg.DBFDir
	log.Printf("[agent] reading DBF files from %s", dbfDir)

	costCenters, err := positouch.ReadCostCenters(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: cost centers: %v", err)
	}

	tenders, err := positouch.ReadTenders(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: tenders: %v", err)
	}

	employees, err := positouch.ReadEmployees(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: employees: %v", err)
	}

	tables, err := positouch.ReadTables(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: tables: %v", err)
	}

	orderTypes, err := positouch.ReadOrderTypes(dbfDir)
	if err != nil {
		log.Printf("[agent] WARNING: order types: %v", err)
	}

	d := cache.Data{
		CostCenters: nilToEmpty(costCenters),
		Tenders:     nilToEmpty(tenders),
		Employees:   nilToEmpty(employees),
		Tables:      nilToEmpty(tables),
		OrderTypes:  nilToEmpty(orderTypes),
	}

	if err := a.cache.Update(d); err != nil {
		log.Printf("[agent] WARNING: updating cache: %v", err)
	} else {
		log.Printf("[agent] cache updated — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d",
			len(d.CostCenters), len(d.Tenders), len(d.Employees), len(d.Tables), len(d.OrderTypes))
	}
}

// nilToEmpty returns an empty (non-nil) slice if s is nil, so that JSON
// serialization produces [] instead of null.
func nilToEmpty(s []map[string]interface{}) []map[string]interface{} {
	if s == nil {
		return []map[string]interface{}{}
	}
	return s
}
