//go:build !windows

package main

// runAsWindowsService is a no-op on non-Windows platforms.
func runAsWindowsService(_ string) (bool, <-chan struct{}) {
	return false, nil
}