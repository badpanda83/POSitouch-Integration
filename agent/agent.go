// Package agent implements the main refresh loop for the POSitouch integration.
// It reads DBF files every 30 minutes and updates the local cache.
package agent

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const refreshInterval = 30 * time.Minute

// Agent orchestrates periodic data refresh from POSitouch DBF files.
type Agent struct {
	cfg    *config.Config
	cache  *cache.Cache
	logger *log.Logger
}

// New creates a new Agent using the supplied configuration.
func New(cfg *config.Config, c *cache.Cache) (*Agent, error) {
	// Set up logging to both file and stdout.
	logFile, err := os.OpenFile(cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Non-fatal: fall back to stdout only.
		fmt.Fprintf(os.Stderr, "warning: cannot open log file %s: %v — logging to stdout only\n", cfg.LogPath, err)
		logFile = os.Stdout
	}

	writer := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(writer, "[POSitouch] ", log.LstdFlags)

	return &Agent{cfg: cfg, cache: c, logger: logger}, nil
}

// Run performs an initial data pull then starts the 30-minute refresh loop.
// It blocks until the provided done channel is closed.
func (a *Agent) Run(done <-chan struct{}) {
	a.logger.Println("Agent starting — initial data pull")
	a.refresh()

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.logger.Println("Scheduled refresh starting")
			a.refresh()
		case <-done:
			a.logger.Println("Agent shutting down")
			return
		}
	}
}

// refresh reads all DBF sources and updates the cache.
func (a *Agent) refresh() {
	dbfDir := a.cfg.DBFDir
	var errs []string

	// Cost Centers
	costCenters, err := positouch.LoadCostCenters(dbfDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("cost_centers: %v", err))
		costCenters = []positouch.CostCenter{}
		a.logger.Printf("WARNING: could not load cost centers: %v", err)
	} else {
		a.logger.Printf("Cost centers loaded: %d records", len(costCenters))
	}

	// Tenders
	tenders, err := positouch.LoadTenders(dbfDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("tenders: %v", err))
		tenders = []positouch.Tender{}
		a.logger.Printf("WARNING: could not load tenders: %v", err)
	} else {
		a.logger.Printf("Tenders loaded: %d records", len(tenders))
	}

	// Employees
	employees, err := positouch.LoadEmployees(dbfDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("employees: %v", err))
		employees = []positouch.Employee{}
		a.logger.Printf("WARNING: could not load employees: %v", err)
	} else {
		a.logger.Printf("Employees loaded: %d records", len(employees))
	}

	// Tables
	tables, err := positouch.LoadTables(dbfDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("tables: %v", err))
		tables = []positouch.Table{}
		a.logger.Printf("WARNING: could not load tables: %v", err)
	} else {
		a.logger.Printf("Tables loaded: %d records", len(tables))
	}

	// Order Types
	orderTypes, err := positouch.LoadOrderTypes(dbfDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("order_types: %v", err))
		orderTypes = []positouch.OrderType{}
		a.logger.Printf("WARNING: could not load order types: %v", err)
	} else {
		a.logger.Printf("Order types loaded: %d records", len(orderTypes))
	}

	if err := a.cache.Update(costCenters, tenders, employees, tables, orderTypes, errs, a.cfg.CachePath); err != nil {
		a.logger.Printf("ERROR: failed to write cache: %v", err)
	} else {
		a.logger.Printf("Cache updated — status: %s, errors: %d", a.cache.Snapshot().Status, len(errs))
	}
}
