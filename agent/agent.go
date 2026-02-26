// Package agent implements the main refresh loop for the POSitouch integration.
// It reads DBF files from the POSitouch installation every 30 minutes and
// updates the local JSON cache.
package agent

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const refreshInterval = 30 * time.Minute

// Agent orchestrates periodic data pulls from POSitouch DBF files.
type Agent struct {
	cfg    *config.Config
	cache  *cache.Cache
	logger *log.Logger
}

// New creates an Agent with its own log file in the install directory.
func New(cfg *config.Config, c *cache.Cache) (*Agent, error) {
	logPath := filepath.Join(cfg.InstallDir, "rooam_agent.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// Fall back to stderr so the agent can still start.
		log.Printf("warning: cannot open log file %s: %v; logging to stderr", logPath, err)
		f = os.Stderr
	}
	logger := log.New(f, "", log.LstdFlags)
	return &Agent{cfg: cfg, cache: c, logger: logger}, nil
}

// Run performs an initial data pull then starts the 30-minute refresh loop.
// It blocks until the context is cancelled (or the process is killed).
func (a *Agent) Run() {
	a.logger.Println("POSitouch integration agent starting")
	a.refresh()

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		a.refresh()
	}
}

// refresh reads all DBF files and updates the cache.
func (a *Agent) refresh() {
	start := time.Now()
	a.logger.Println("refresh: starting")

	data := cache.Data{
		LastUpdated: time.Now(),
		Status:      "ok",
		Errors:      []string{},
	}

	// Cost Centers
	cc, err := positouch.ReadCostCenters(a.cfg.DBFDir)
	if err != nil {
		a.logger.Printf("refresh: cost centers: %v", err)
		data.Errors = append(data.Errors, fmt.Sprintf("cost_centers: %v", err))
	} else {
		data.CostCenters = cc
		a.logger.Printf("refresh: cost centers: %d records", len(cc))
	}

	// Tenders
	tenders, err := positouch.ReadTenders(a.cfg.DBFDir)
	if err != nil {
		a.logger.Printf("refresh: tenders: %v", err)
		data.Errors = append(data.Errors, fmt.Sprintf("tenders: %v", err))
	} else {
		data.Tenders = tenders
		a.logger.Printf("refresh: tenders: %d records", len(tenders))
	}

	// Employees
	employees, err := positouch.ReadEmployees(a.cfg.DBFDir)
	if err != nil {
		a.logger.Printf("refresh: employees: %v", err)
		data.Errors = append(data.Errors, fmt.Sprintf("employees: %v", err))
	} else {
		data.Employees = employees
		a.logger.Printf("refresh: employees: %d records", len(employees))
	}

	// Tables
	tables, err := positouch.ReadTables(a.cfg.DBFDir)
	if err != nil {
		a.logger.Printf("refresh: tables: %v", err)
		data.Errors = append(data.Errors, fmt.Sprintf("tables: %v", err))
	} else {
		data.Tables = tables
		a.logger.Printf("refresh: tables: %d records", len(tables))
	}

	// Order Types
	orderTypes, err := positouch.ReadOrderTypes(a.cfg.DBFDir)
	if err != nil {
		a.logger.Printf("refresh: order types: %v", err)
		data.Errors = append(data.Errors, fmt.Sprintf("order_types: %v", err))
	} else {
		data.OrderTypes = orderTypes
		a.logger.Printf("refresh: order types: %d records", len(orderTypes))
	}

	if len(data.Errors) > 0 {
		data.Status = "partial"
	}

	// Ensure nil slices become empty JSON arrays.
	if data.CostCenters == nil {
		data.CostCenters = []positouch.CostCenter{}
	}
	if data.Tenders == nil {
		data.Tenders = []positouch.Tender{}
	}
	if data.Employees == nil {
		data.Employees = []positouch.Employee{}
	}
	if data.Tables == nil {
		data.Tables = []positouch.Table{}
	}
	if data.OrderTypes == nil {
		data.OrderTypes = []positouch.OrderType{}
	}

	if err := a.cache.Update(data); err != nil {
		a.logger.Printf("refresh: cache update failed: %v", err)
	}

	a.logger.Printf("refresh: completed in %s (status=%s, errors=%d)",
		time.Since(start).Round(time.Millisecond), data.Status, len(data.Errors))
}
