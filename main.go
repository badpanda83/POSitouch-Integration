// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and persists it to rooam_cache.json.
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
	configPath := flag.String("config", config.DefaultConfigPath, "path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	log.Printf("[main] loading config from %s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[main] failed to load config: %v", err)
	}

	log.Printf("[main] install dir : %s", cfg.InstallDir)
	log.Printf("[main] SC dir      : %s", cfg.SCDir)
	log.Printf("[main] DBF dir     : %s", cfg.DBFDir)
	log.Printf("[main] ALTDBF dir  : %s", cfg.AltDBFDir)

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
	log.Println("[main] goodbye")
}
