package main

import (
    "fmt"
    "os/exec"
    "os"
    "time"
)

// ExportConfig holds settings for export automation
type ExportConfig struct {
    WexportPath     string // path to wexport.exe
    ExportType      string // e.g., "EXPORTMENU"
    ExportDirectory string // where the .EXP file will be saved
}

// ExportPOSitouchMenu calls wexport.exe to perform menu export
func ExportPOSitouchMenu(cfg ExportConfig) error {
    // Make sure export directory exists
    if err := os.MkdirAll(cfg.ExportDirectory, os.ModePerm); err != nil {
        return fmt.Errorf("could not create export directory: %v", err)
    }

    // Compose full export command
    // Adapt these args if your POS version requires different flags!
    args := []string{
        cfg.ExportType,
    }
    cmd := exec.Command(cfg.WexportPath, args...)
    cmd.Dir = cfg.ExportDirectory

    fmt.Printf("[export] Running: %s %v\n", cfg.WexportPath, args)
    out, err := cmd.CombinedOutput()
    fmt.Printf("[export] Output:\n%s\n", string(out))
    if err != nil {
        return fmt.Errorf("wexport.exe failed: %v", err)
    }
    // Wait briefly for export to finish (can remove if not needed)
    time.Sleep(2 * time.Second)
    return nil
}

// This is a simple test main. Adapt the config as needed.
func main() {
    cfg := ExportConfig{
        WexportPath:     `C:\SC\WExport.EXE`,  // Adjust to actual path!
        ExportType:      "EXPORTMENU",          // Or what your install needs; check docs
        ExportDirectory: `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export`,       // Set as needed
    }

    err := ExportPOSitouchMenu(cfg)
    if err != nil {
        fmt.Printf("Export failed: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("Menu export completed successfully!")
}