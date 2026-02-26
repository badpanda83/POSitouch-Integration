// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes
// and stores the extracted data in a local JSON cache file.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "Path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	printBanner()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("main: failed to load config: %v", err)
	}
	log.Printf("main: config loaded — location=%q dbfDir=%q scDir=%q",
		cfg.Location.Name, cfg.DBFDir(), cfg.SCDir())

	// Initialise cache
	c := cache.New(cfg.ConfigDir())
	if err := c.LoadFromDisk(); err != nil {
		log.Printf("main: warning — could not load cache from disk: %v", err)
	} else {
		snap := c.Get()
		log.Printf("main: cache loaded (last_updated=%s cost_centers=%d tenders=%d employees=%d tables=%d order_types=%d)",
			snap.LastUpdated.Format("2006-01-02 15:04:05"),
			len(snap.CostCenters), len(snap.Tenders), len(snap.Employees),
			len(snap.Tables), len(snap.OrderTypes))
	}

	// Start agent
	a := agent.New(cfg, c)
	a.Start()

	// Wait for OS signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("main: shutdown signal received — stopping agent")
	a.Stop()
	log.Println("main: shutdown complete")
}

func printBanner() {
	log.Println("============================================")
	log.Println("  POSitouch Integration Agent")
	log.Println("  Rooam — POSitouch DBF Reader")
	log.Println("============================================")
}
