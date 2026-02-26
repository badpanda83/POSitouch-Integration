// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and persists it to rooam_cache.json.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

const (
	appName    = "rooam-pos-agent"
	appVersion = "1.0.0"
)

func main() {
	// --- Updated for config compatibility ---
	configPath := flag.String("config", config.DefaultConfigPath, "path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	fmt.Printf("╔══════════════════════════════════════════╗\n")
	fmt.Printf("║  %-38s║\n", fmt.Sprintf("%s v%s", appName, appVersion))
	fmt.Printf("║  %-38s║\n", "POSitouch Integration Agent")
	fmt.Printf("╚══════════════════════════════════════════╝\n\n")

	log.Printf("[main] config path : %s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[main] failed to load config: %v", err)
	}

	log.Printf("[main] location    : %s", cfg.Location.Name)
	log.Printf("[main] install dir : %s", cfg.InstallDir)
	log.Printf("[main] SC dir      : %s", cfg.SCDir)
	log.Printf("[main] DBF dir     : %s", cfg.DBFDir)
	log.Printf("[main] ALTDBF dir  : %s", cfg.AltDBFDir) // Note: matches Config struct field

	// --- Updated: pass correct directory for cache file ---
	c := cache.New(cfg.InstallDir)
	a := agent.New(cfg, c)

	// Handle OS signals for graceful shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("[main] received signal %s — shutting down", sig)
		a.Stop()
	}()

	a.Start() // blocks until Stop() is called
	log.Println("[main] Agent stopped")
}