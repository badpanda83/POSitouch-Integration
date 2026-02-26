// main is the entry point for the POSitouch integration agent.
//
// Usage:
//
//	positouch-agent [install-dir]
//
// If install-dir is omitted it defaults to:
//
//	C:\Program Files\Rooam\POSitouch   (Windows)
//	./                                 (other platforms, for testing)
package main

import (
	"log"
	"os"
	"runtime"

	"github.com/badpanda83/POSitouch-Integration/agent"
	"github.com/badpanda83/POSitouch-Integration/cache"
	"github.com/badpanda83/POSitouch-Integration/config"
)

func main() {
	installDir := defaultInstallDir()
	if len(os.Args) > 1 {
		installDir = os.Args[1]
	}

	cfg, err := config.Load(installDir)
	if err != nil {
		log.Fatalf("failed to load config from %s: %v", installDir, err)
	}

	c := cache.New(installDir)

	a, err := agent.New(cfg, c)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	a.Run()
}

func defaultInstallDir() string {
	if runtime.GOOS == "windows" {
		return config.DefaultInstallDir
	}
	// On non-Windows (CI, development), use the current directory.
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
