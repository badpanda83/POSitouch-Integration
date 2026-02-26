// main is the entry point for the POSitouch integration agent.
// It loads the configuration, performs an initial data pull, then runs
// a background refresh loop that updates the local cache every 30 minutes.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: could not load config: %v\n", err)
		os.Exit(1)
	}

	c := cache.New()

	a, err := agent.New(cfg, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: could not initialise agent: %v\n", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	done := make(chan struct{})
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		close(done)
	}()

	a.Run(done)
}
