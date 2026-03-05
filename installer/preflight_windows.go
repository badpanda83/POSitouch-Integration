//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// checkAdminPrivileges attempts to open a registry key that requires
// administrator rights. It prints a warning if the process is not elevated.
func checkAdminPrivileges() {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services`, registry.WRITE)
	if err != nil {
		fmt.Println("[preflight] ⚠ admin check: not running as administrator — service installation will fail")
		return
	}
	key.Close()
	fmt.Println("[preflight] ✓ admin check: running as administrator")
}
