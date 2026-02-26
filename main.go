package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

const (
	version  = "1.0.0"
	interval = 30 * time.Minute
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)

	defaultConfig := findDefaultConfig()
	configPath := flag.String("config", defaultConfig, "path to rooam_config.json")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] load config: %v", err)
	}

	cacheFile := filepath.Join(filepath.Dir(*configPath), "rooam_cache.json")
	c := cache.NewCache(cacheFile)
	if err := c.Load(); err != nil {
		log.Printf("[WARN] load cache: %v", err)
	}

	printBanner(cfg)

	a := agent.NewAgent(cfg, c, interval)
	a.Start()

	// Wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[INFO] shutting down…")
	a.Stop()
}

func printBanner(cfg *config.RooamConfig) {
	fmt.Println("═══════════════════════════════════════════════")
	fmt.Printf("  Rooam POSitouch Integration Agent v%s\n", version)
	fmt.Printf("  Location: %s\n", cfg.Location.Name)
	fmt.Printf("  SC Path:  %s\n", cfg.SCDir())
	fmt.Printf("  DBF Path: %s\n", cfg.DBFDir())
	fmt.Printf("  Interval: 30 minutes\n")
	fmt.Println("═══════════════════════════════════════════════")
}

// findDefaultConfig looks for rooam_config.json in the current directory
// and then in the standard install location.
func findDefaultConfig() string {
	if _, err := os.Stat("rooam_config.json"); err == nil {
		return "rooam_config.json"
	}
	return `C:\Program Files (x86)\Rooam\POSitouch\rooam_config.json`
}
