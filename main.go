// POSitouch Integration Agent — reads POSitouch DBF files every 30 minutes,
// caches the data in memory, and exposes it via REST endpoints.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/badpanda83/POSitouch-Integration/config"
)

const (
	appName    = "rooam-pos-agent"
	appVersion = "1.0.0"
	exportDir  = `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export`
	tablesXML  = exportDir + `\set1.xml`
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath, "path to rooam_config.json")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	fmt.Printf("╔═══════════════════════════════════════════╗\n")
	fmt.Printf("║  %-38s║\n", fmt.Sprintf("%s v%s", appName, appVersion))
	fmt.Printf("║  %-38s║\n", "POSitouch Integration Agent")
	fmt.Printf("╚═══════════════════════════════════════════╝\n\n")

	// If started by the Windows SCM, runAsWindowsService launches runAgent in a
	// goroutine, signals Running to the SCM, then blocks until Stop/Shutdown.
	// It returns true and main() exits cleanly.
	// Interactive runs return false and fall through to the interactive path below.
	if runAsWindowsService(*configPath) {
		return
	}

	// --- Interactive mode ---
	// Create a stop channel that is closed when the user presses Ctrl-C or
	// sends SIGTERM. runAgent selects on this channel to shut down cleanly.
	stop := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("[main] received OS signal — shutting down")
		close(stop)
	}()

	runAgent(*configPath, stop)
}