// rooam-pos-agent is a background integration agent that reads POSitouch DBF
// files and stores the extracted data in a local JSON cache.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rooam-pos-agent/agent"
	"rooam-pos-agent/cache"
	"rooam-pos-agent/config"
)

func main() {
	configPath := flag.String("config", "rooam_config.json", "path to rooam_config.json")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	printBanner(cfg)

	c := cache.New()

	// Determine the cache file path — same directory as the config file.
	cacheDir := filepath.Dir(*configPath)
	cachePath := filepath.Join(cacheDir, "rooam_cache.json")

	// Load any existing cache from disk (best-effort).
	if err := c.LoadFromFile(cachePath); err != nil {
		log.Printf("[main] no existing cache loaded (%v)", err)
	}

	a := agent.New(cfg, c, cachePath)
	a.Start()

	// Wait for OS termination signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[main] received signal %s — shutting down", sig)

	a.Stop()

	// Final cache save on shutdown.
	if err := c.SaveToFile(cachePath); err != nil {
		log.Printf("[main] warning: final cache save failed: %v", err)
	} else {
		log.Printf("[main] cache saved to %s", cachePath)
	}

	log.Println("[main] shutdown complete")
}

func printBanner(cfg *config.Config) {
	loc := cfg.Location
	fmt.Println("========================================")
	fmt.Println("  Rooam POSitouch Integration Agent")
	fmt.Println("========================================")
	if loc.Name != "" {
		fmt.Printf("  Venue   : %s\n", loc.Name)
	}
	if loc.Address1 != "" {
		fmt.Printf("  Address : %s\n", loc.Address1)
	}
	if loc.Address2 != "" {
		fmt.Printf("            %s\n", loc.Address2)
	}
	if loc.City != "" || loc.State != "" || loc.Zip != "" {
		fmt.Printf("            %s, %s %s\n", loc.City, loc.State, loc.Zip)
	}
	if loc.Phone != "" {
		fmt.Printf("  Phone   : %s\n", loc.Phone)
	}
	if loc.Email != "" {
		fmt.Printf("  Email   : %s\n", loc.Email)
	}
	fmt.Println("----------------------------------------")
	fmt.Printf("  SC Path     : %s\n", cfg.SCPath)
	fmt.Printf("  DBF Path    : %s\n", cfg.DBFPath)
	fmt.Printf("  ALTDBF Path : %s\n", cfg.ALTDBFPath)
	fmt.Println("========================================")
}
