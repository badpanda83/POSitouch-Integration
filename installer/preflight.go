package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/badpanda83/POSitouch-Integration/config"
)

// runPreflight performs pre-installation checks. cfg may be nil for a partial
// check before the config file exists.
func runPreflight(cfg *config.Config) error {
	var errs []string

	// 1. OS check — must be Windows.
	if runtime.GOOS != "windows" {
		fmt.Printf("[preflight] ✗ OS check: must be Windows (got %s)\n", runtime.GOOS)
		errs = append(errs, fmt.Sprintf("OS check failed: requires Windows, got %s", runtime.GOOS))
	} else {
		fmt.Println("[preflight] ✓ OS check: Windows")
	}

	// 2. Running as administrator (Windows-only, handled in preflight_windows.go).
	checkAdminPrivileges()

	if cfg == nil {
		if len(errs) > 0 {
			return errors.New(joinErrors(errs))
		}
		return nil
	}

	// 3. Agent binary exists.
	agentExe := findAgentBinary(cfg)
	if agentExe == "" {
		fmt.Println("[preflight] ✗ agent binary: rooam-pos-agent.exe not found")
		errs = append(errs, "rooam-pos-agent.exe not found next to installer or in install dir")
	} else {
		fmt.Printf("[preflight] ✓ agent binary: %s\n", agentExe)
	}

	// 4. Config file readable (if it already exists).
	if _, err := os.Stat(config.DefaultConfigPath); err == nil {
		if _, err := config.Load(config.DefaultConfigPath); err != nil {
			fmt.Printf("[preflight] ✗ config readable: %v\n", err)
			errs = append(errs, fmt.Sprintf("config parse error: %v", err))
		} else {
			fmt.Printf("[preflight] ✓ config readable: %s\n", config.DefaultConfigPath)
		}
	}

	posType := cfg.EffectivePOSType()

	if posType == "positouch" {
		// 5. XML directories.
		if err := checkDirReadable(cfg.XMLDir); err != nil {
			fmt.Printf("[preflight] ✗ xml_dir (%s): %v\n", cfg.XMLDir, err)
			errs = append(errs, fmt.Sprintf("xml_dir: %v", err))
		} else {
			fmt.Printf("[preflight] ✓ xml_dir: %s\n", cfg.XMLDir)
		}

		if err := checkDirReadable(cfg.XMLCloseDir); err != nil {
			fmt.Printf("[preflight] ✗ xml_close_dir (%s): %v\n", cfg.XMLCloseDir, err)
			errs = append(errs, fmt.Sprintf("xml_close_dir: %v", err))
		} else {
			fmt.Printf("[preflight] ✓ xml_close_dir: %s\n", cfg.XMLCloseDir)
		}

		if err := checkDirWritable(cfg.XMLInOrderDir); err != nil {
			fmt.Printf("[preflight] ✗ xml_inorder_dir (%s): %v\n", cfg.XMLInOrderDir, err)
			errs = append(errs, fmt.Sprintf("xml_inorder_dir: %v", err))
		} else {
			fmt.Printf("[preflight] ✓ xml_inorder_dir: %s\n", cfg.XMLInOrderDir)
		}

		// 6. DBF directory.
		if err := checkDirReadable(cfg.DBFDir); err != nil {
			fmt.Printf("[preflight] ✗ dbf_dir (%s): %v\n", cfg.DBFDir, err)
			errs = append(errs, fmt.Sprintf("dbf_dir: %v", err))
		} else {
			fmt.Printf("[preflight] ✓ dbf_dir: %s\n", cfg.DBFDir)
		}
	}

	// 7. MICROS 3700 config present.
	if posType == "micros3700" {
		if cfg.MICROS3700 == nil || cfg.MICROS3700.TransactionServicesURL == "" {
			fmt.Println("[preflight] ✗ MICROS 3700 config: micros3700 config missing or transaction_services_url is empty")
			errs = append(errs, "micros3700 config missing or transaction_services_url empty")
		} else {
			fmt.Printf("[preflight] ✓ MICROS 3700 config: %s\n", cfg.MICROS3700.TransactionServicesURL)
		}
	}

	if len(errs) > 0 {
		return errors.New(joinErrors(errs))
	}
	return nil
}

// findAgentBinary looks for rooam-pos-agent.exe next to the installer and in cfg.InstallDir.
func findAgentBinary(cfg *config.Config) string {
	candidates := []string{
		"rooam-pos-agent.exe",
	}
	if cfg.InstallDir != "" {
		candidates = append(candidates, filepath.Join(cfg.InstallDir, "rooam-pos-agent.exe"))
	}
	exe, err := os.Executable()
	if err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "rooam-pos-agent.exe"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func checkDirReadable(dir string) error {
	if dir == "" {
		return fmt.Errorf("path is empty")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func checkDirWritable(dir string) error {
	if err := checkDirReadable(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".rooam-preflight-*")
	if err != nil {
		return fmt.Errorf("not writable: %w", err)
	}
	tmp.Close()
	_ = os.Remove(tmp.Name())
	return nil
}

func joinErrors(errs []string) string {
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += "; "
		}
		result += e
	}
	return result
}
