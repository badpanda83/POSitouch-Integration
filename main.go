// main.go is the entry point for the POSitouch integration agent.
// It loads configuration, starts the agent, and waits for an OS shutdown signal.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	configPath := config.DefaultConfigPath
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	log.Printf("main: loading config from %s", configPath)
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("main: failed to load config: %v", err)
	}

	log.Printf("main: location   = %s", cfg.Location.Name)
	log.Printf("main: DBF path   = %s", cfg.DBFPath())
	log.Printf("main: SC path    = %s", cfg.SCPath())
	log.Printf("main: tender ID  = %s", cfg.Rooam.TenderID)
	log.Printf("main: employee ID = %s", cfg.Rooam.EmployeeID)

	a := agent.New(cfg, configPath)
	a.Start()

	// Block until SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("main: shutdown signal received")
	a.Stop()
}
