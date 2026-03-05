//go:build !windows

package main

import "fmt"

func installService(agentPath, configPath string) error {
	return fmt.Errorf("service installation is only supported on Windows")
}

func uninstallService() error {
	return fmt.Errorf("service removal is only supported on Windows")
}

func serviceStatus() (string, error) {
	return "not supported on this platform", nil
}
