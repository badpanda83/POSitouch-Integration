package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/badpanda83/POSitouch-Integration/config"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "install":
		runInstall()
	case "uninstall":
		runUninstall()
	case "configure":
		runConfigure()
	case "check":
		runCheck()
	case "activate":
		if err := runActivate(); err != nil {
			fmt.Fprintf(os.Stderr, "activate: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("rooam-pos-installer version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: installer.exe <command>

Commands:
  install     Install and register the POS agent as a Windows Service
  uninstall   Stop and remove the POS agent Windows Service
  configure   Re-run the configuration wizard
  check       Run pre-flight and connectivity checks only
  activate    Activate OAuth credentials (requires IdP — not yet available)
  version     Print version information`)
}

// runInstall runs the full installation flow:
//  1. preflight (partial), 2. wizard, 3. preflight (full),
//  4. connectivity check, 5. optional service install.
func runInstall() {
	printBanner()

	// Step 1: partial preflight (before config exists).
	fmt.Println()
	fmt.Println("--- Pre-flight checks (initial) ---")
	_ = runPreflight(nil)

	// Step 2: run wizard to generate rooam_config.json.
	fmt.Println()
	cfg, err := runWizard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration wizard failed: %v\n", err)
		os.Exit(1)
	}

	// Step 3: full preflight with config.
	fmt.Println()
	fmt.Println("--- Pre-flight checks (full) ---")
	if err := runPreflight(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[preflight] errors: %v\n", err)
		// Non-fatal — continue.
	}

	// Step 4: connectivity check.
	fmt.Println()
	fmt.Println("--- Connectivity checks ---")
	if err := runConnectivityCheck(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[connectivity] errors: %v\n", err)
		// Non-fatal — user may want to install anyway.
	}

	// Step 5: ask to install as Windows Service.
	fmt.Println()
	fmt.Print("Install as Windows Service? [Y/n]: ")
	scanner := bufio.NewScanner(os.Stdin)
	answer := "y"
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			answer = strings.ToLower(line)
		}
	}
	if answer == "y" || answer == "yes" {
		agentExe := findAgentBinary(cfg)
		if agentExe == "" {
			// Try default name — the service binary path needs to be absolute.
			exe, err := os.Executable()
			if err == nil {
				agentExe = filepath.Join(filepath.Dir(exe), "rooam-pos-agent.exe")
			} else {
				agentExe = "rooam-pos-agent.exe"
			}
		}
		configPath, err := filepath.Abs(config.DefaultConfigPath)
		if err != nil {
			configPath = config.DefaultConfigPath
		}
		if err := installService(agentExe, configPath); err != nil {
			fmt.Fprintf(os.Stderr, "service install failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println()
	fmt.Println("=== Installation complete ===")
}

// runUninstall stops and removes the Windows service, then optionally deletes config.
func runUninstall() {
	if err := uninstallService(); err != nil {
		fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("Delete rooam_config.json? [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		if strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
			if err := os.Remove(config.DefaultConfigPath); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: could not delete config: %v\n", err)
			} else {
				fmt.Println("[uninstall] Config file deleted")
			}
		}
	}
}

// runConfigure re-runs the wizard to update rooam_config.json.
func runConfigure() {
	if _, err := runWizard(); err != nil {
		fmt.Fprintf(os.Stderr, "configure: %v\n", err)
		os.Exit(1)
	}
}

// runCheck runs preflight and connectivity checks without installing anything.
func runCheck() {
	fmt.Println("--- Pre-flight checks ---")
	_ = runPreflight(nil)

	cfg, err := config.Load(config.DefaultConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[check] could not load config (%s): %v\n", config.DefaultConfigPath, err)
		fmt.Println("[check] Run 'installer.exe configure' to generate a config file.")
		os.Exit(1)
	}

	if err := runPreflight(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[preflight] errors: %v\n", err)
	}

	fmt.Println()
	fmt.Println("--- Connectivity checks ---")
	if err := runConnectivityCheck(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "[connectivity] errors: %v\n", err)
		os.Exit(1)
	}
}

func printBanner() {
	fmt.Printf("╔══════════════════════════════════════╗\n")
	fmt.Printf("║  Rooam POS Agent Installer v%-8s ║\n", version)
	fmt.Printf("╚══════════════════════════════════════╝\n")
}
