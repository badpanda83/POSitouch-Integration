// rooam-pos-agent — POSitouch integration agent.
// Reads POSitouch DBF files every 30 minutes and caches the data locally.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"rooam-pos-agent/agent"
	"rooam-pos-agent/cache"
	"rooam-pos-agent/config"
)

const (
	appName    = "rooam-pos-agent"
	appVersion = "1.0.0"
)

func main() {
	configPath := flag.String("config", "rooam_config.json", "path to rooam_config.json")
	flag.Parse()

	fmt.Printf("╔══════════════════════════════════════╗\n")
	fmt.Printf("║  %s  v%s               ║\n", appName, appVersion)
	fmt.Printf("║  POSitouch Integration Agent          ║\n")
	fmt.Printf("╚══════════════════════════════════════╝\n\n")

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("fatal: load config: %v", err)
	}
	log.Printf("config loaded — location: %s", cfg.Location.Name)
	log.Printf("  SC path:     %s", cfg.SCPath)
	log.Printf("  DBF path:    %s", cfg.DBFPath)
	log.Printf("  ALTDBF path: %s", cfg.ALTDBFPath)

	// Initialize cache
	c, err := cache.New("rooam_cache.json")
	if err != nil {
		log.Fatalf("fatal: init cache: %v", err)
	}

	// Start agent
	a := agent.New(cfg, c)
	a.Start()

	// Wait for OS termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %s, shutting down…", sig)

	a.Stop()

	if err := c.Save(); err != nil {
		log.Printf("warning: final cache save failed: %v", err)
	}

	log.Println("goodbye.")
}
