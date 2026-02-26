// Package agent implements the main polling loop that reads POSitouch DBF files
// and keeps the cache up to date.
package agent

import (
	"log"
	"time"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/config"
	"rooam-pos-agent/positouch"
)

const pollInterval = 30 * time.Minute

// Agent orchestrates periodic pulls from the POSitouch DBF files.
type Agent struct {
	cfg    *config.Config
	cache  *cache.Cache
	ticker *time.Ticker
	stop   chan struct{}
}

// New creates an Agent bound to the given config and cache.
func New(cfg *config.Config, c *cache.Cache) *Agent {
	return &Agent{
		cfg:   cfg,
		cache: c,
		stop:  make(chan struct{}),
	}
}

// Start performs an immediate pull and then schedules a pull every 30 minutes.
// It returns immediately; the polling loop runs in a background goroutine.
func (a *Agent) Start() {
	log.Println("agent: starting — performing initial pull")
	a.Pull()

	a.ticker = time.NewTicker(pollInterval)
	go func() {
		for {
			select {
			case <-a.ticker.C:
				log.Println("agent: scheduled pull triggered")
				a.Pull()
			case <-a.stop:
				return
			}
		}
	}()
}

// Stop cancels the polling loop and releases resources.
func (a *Agent) Stop() {
	log.Println("agent: stopping")
	if a.ticker != nil {
		a.ticker.Stop()
	}
	close(a.stop)
}

// Pull reads all five data types from the POSitouch DBF files, updates the
// cache and persists it to disk.  Individual file-read errors are logged but do
// not abort the pull.
func (a *Agent) Pull() {
	log.Println("agent: pulling POSitouch data")

	costCenters, err := positouch.ReadCostCenters(a.cfg)
	if err != nil {
		log.Printf("agent: cost centers: %v", err)
		costCenters = nil
	} else {
		log.Printf("agent: cost centers: %d records", len(costCenters))
	}

	tenders, err := positouch.ReadTenders(a.cfg)
	if err != nil {
		log.Printf("agent: tenders: %v", err)
		tenders = nil
	} else {
		log.Printf("agent: tenders: %d records", len(tenders))
	}

	employees, err := positouch.ReadEmployees(a.cfg)
	if err != nil {
		log.Printf("agent: employees: %v", err)
		employees = nil
	} else {
		log.Printf("agent: employees: %d records", len(employees))
	}

	tables, err := positouch.ReadTables(a.cfg)
	if err != nil {
		log.Printf("agent: tables: %v", err)
		tables = nil
	} else {
		log.Printf("agent: tables: %d records", len(tables))
	}

	orderTypes, err := positouch.ReadOrderTypes(a.cfg)
	if err != nil {
		log.Printf("agent: order types: %v", err)
		orderTypes = nil
	} else {
		log.Printf("agent: order types: %d records", len(orderTypes))
	}

	a.cache.Update(costCenters, tenders, employees, tables, orderTypes)

	if err := a.cache.Save(); err != nil {
		log.Printf("agent: saving cache: %v", err)
	} else {
		log.Println("agent: cache saved successfully")
	}
}
