// Package agent implements the main polling loop that refreshes POSitouch data.
package agent

import (
	"log"
	"time"

	"rooam-pos-agent/cache"
	"rooam-pos-agent/config"
	"rooam-pos-agent/positouch"
)

const pollInterval = 30 * time.Minute

// Agent periodically reads POSitouch DBF files and updates the cache.
type Agent struct {
	cfg       *config.Config
	cache     *cache.Cache
	cachePath string
	stop      chan struct{}
	done      chan struct{}
}

// New creates a new Agent.
func New(cfg *config.Config, c *cache.Cache, cachePath string) *Agent {
	return &Agent{
		cfg:       cfg,
		cache:     c,
		cachePath: cachePath,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
}

// Start runs an immediate data pull, then polls on a 30-minute ticker.
// It returns immediately; the polling loop runs in a background goroutine.
func (a *Agent) Start() {
	go a.run()
}

// Stop signals the agent to stop and waits for the background goroutine to exit.
func (a *Agent) Stop() {
	close(a.stop)
	<-a.done
}

func (a *Agent) run() {
	defer close(a.done)

	// Immediate first pull on startup.
	a.pull()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.pull()
		case <-a.stop:
			log.Println("[agent] stopping")
			return
		}
	}
}

// pull reads all data sources, updates the cache, and persists to disk.
// Individual reader failures are logged but do not abort the pull.
func (a *Agent) pull() {
	start := time.Now()
	log.Printf("[agent] starting data pull at %s", start.Format(time.RFC3339))

	data := &cache.CacheData{
		LastUpdated: start,
	}

	data.CostCenters = positouch.ReadCostCenters(a.cfg.DBFPath)
	data.Tenders = positouch.ReadTenders(a.cfg.DBFPath)
	data.Employees = positouch.ReadEmployees(a.cfg.DBFPath, a.cfg.SCPath)
	data.Tables = positouch.ReadTables(a.cfg.DBFPath)
	data.OrderTypes = positouch.ReadOrderTypes(a.cfg.SCPath)

	a.cache.Update(data)

	if err := a.cache.SaveToFile(a.cachePath); err != nil {
		log.Printf("[agent] warning: could not save cache to %q: %v", a.cachePath, err)
	}

	log.Printf("[agent] pull complete in %s — cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d",
		time.Since(start).Round(time.Millisecond),
		len(data.CostCenters),
		len(data.Tenders),
		len(data.Employees),
		len(data.Tables),
		len(data.OrderTypes),
	)
}
