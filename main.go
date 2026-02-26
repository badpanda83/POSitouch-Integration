// rooam-pos-agent is the POSitouch integration agent.
// It reads POSitouch DBF files, caches the data locally, and refreshes every
// 30 minutes.
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

const version = "1.0.0"

func main() {
	configPath := flag.String("config", "rooam_config.json", "path to rooam_config.json")
	flag.Parse()

	fmt.Printf("╔══════════════════════════════════════╗\n")
	fmt.Printf("║  Rooam POSitouch Integration Agent   ║\n")
	fmt.Printf("║  Version %-28s║\n", version)
	fmt.Printf("╚══════════════════════════════════════╝\n\n")

	// Load configuration.
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	log.Printf("config loaded: location=%s, SC=%s, DBF=%s, ALTDBF=%s",
		cfg.Location.Name, cfg.SCDir, cfg.DBFDir, cfg.ALTDBFDir)

	// Initialise cache (warm start from disk if available).
	cacheFile := "rooam_cache.json"
	c := cache.New(cacheFile)
	if err := c.Load(); err != nil {
		log.Printf("warning: could not load cache from disk: %v", err)
	}

	// Create and start the agent.
	a := agent.New(cfg, c)
	a.Start()

	// Wait for SIGINT or SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received signal %v — shutting down", sig)
	a.Stop()
	log.Println("goodbye")
}
