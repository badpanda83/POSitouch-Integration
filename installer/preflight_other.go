//go:build !windows

package main

import "fmt"

// checkAdminPrivileges is a no-op stub on non-Windows platforms.
func checkAdminPrivileges() {
	fmt.Println("[preflight] ⚠ admin check: skipped (non-Windows platform)")
}
