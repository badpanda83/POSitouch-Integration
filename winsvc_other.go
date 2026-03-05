//go:build !windows

// winsvc_other.go — no-op stubs so the package compiles on Linux / macOS.

package main

// runAsWindowsService always returns (false, nil) on non-Windows platforms.
func runAsWindowsService(_ string) (bool, <-chan struct{}) { return false, nil }