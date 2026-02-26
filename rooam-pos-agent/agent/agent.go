// Package agent implements the main polling loop that reads POSitouch DBF files
// every 30 minutes and updates the local cache.
package agent

import (
	"log"
	"time"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/config"
	"rooam-pos-agent/positouch"
)

const refreshInterval = 30 * time.Minute

// Agent polls POSitouch DBF files on a fixed interval and keeps the cache
// up-to-date.
type Agent struct {
	cfg    *config.Config
	cache  *cache.Cache
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new Agent with the given configuration and cache.
func New(cfg *config.Config, c *cache.Cache) *Agent {
	return &Agent{
		cfg:    cfg,
		cache:  c,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start performs an immediate data pull and then schedules refreshes every 30
// minutes.  It runs in its own goroutine.
func (a *Agent) Start() {
	go a.run()
}

// Stop signals the agent to stop after the current refresh completes and waits
// for the goroutine to exit.
func (a *Agent) Stop() {
	close(a.stopCh)
	<-a.doneCh
}

func (a *Agent) run() {
	defer close(a.doneCh)

	// Immediate first pull
	a.refresh()

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.refresh()
		case <-a.stopCh:
			log.Println("agent: stopping")
			return
		}
	}
}

// refresh reads all POSitouch DBF files and updates the cache.
func (a *Agent) refresh() {
	start := time.Now()
	log.Printf("agent: starting refresh at %s", start.Format(time.RFC3339))

	costCenters := safeReadCostCenters(a.cfg.DBFPath)
	tenders := safeReadTenders(a.cfg.DBFPath)
	employees := safeReadEmployees(a.cfg.DBFPath, a.cfg.SCPath)
	tables := safeReadTables(a.cfg.DBFPath)
	orderTypes := safeReadOrderTypes(a.cfg.SCPath)

	a.cache.Update(costCenters, tenders, employees, tables, orderTypes)

	if err := a.cache.Save(); err != nil {
		log.Printf("agent: warning: failed to save cache: %v", err)
	}

	log.Printf("agent: refresh complete in %s — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d",
		time.Since(start).Round(time.Millisecond),
		len(costCenters),
		len(tenders),
		len(employees),
		len(tables),
		len(orderTypes),
	)
}

func safeReadCostCenters(dbfPath string) []positouch.CostCenter {
	result, err := positouch.ReadCostCenters(dbfPath)
	if err != nil {
		log.Printf("agent: warning reading cost centers: %v", err)
		return []positouch.CostCenter{}
	}
	return result
}

func safeReadTenders(dbfPath string) []positouch.Tender {
	result, err := positouch.ReadTenders(dbfPath)
	if err != nil {
		log.Printf("agent: warning reading tenders: %v", err)
		return []positouch.Tender{}
	}
	return result
}

func safeReadEmployees(dbfPath, scPath string) []positouch.Employee {
	result, err := positouch.ReadEmployees(dbfPath, scPath)
	if err != nil {
		log.Printf("agent: warning reading employees: %v", err)
		return []positouch.Employee{}
	}
	return result
}

func safeReadTables(dbfPath string) []positouch.Table {
	result, err := positouch.ReadTables(dbfPath)
	if err != nil {
		log.Printf("agent: warning reading tables: %v", err)
		return []positouch.Table{}
	}
	return result
}

func safeReadOrderTypes(scPath string) []positouch.OrderType {
	result, err := positouch.ReadOrderTypes(scPath)
	if err != nil {
		log.Printf("agent: warning reading order types: %v", err)
		return []positouch.OrderType{}
	}
	return result
}
