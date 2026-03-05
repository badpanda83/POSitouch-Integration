//go:build windows

package main

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "RooamPOSAgent"
const serviceDisplayName = "Rooam POS Agent"
const serviceDescription = "Rooam POS Integration Agent — syncs POS data to the cloud"

// installService creates and starts the Windows service.
func installService(agentPath, configPath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service: connect to SCM: %w", err)
	}
	defer m.Disconnect()

	// Quote both paths to handle spaces in directory names.
	binaryPathName := fmt.Sprintf(`"%s" -config "%s"`, agentPath, configPath)

	s, err := m.CreateService(serviceName, binaryPathName, mgr.Config{
		DisplayName: serviceDisplayName,
		StartType:   mgr.StartAutomatic,
		ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
	})
	if err != nil {
		return fmt.Errorf("service: create service: %w", err)
	}
	defer s.Close()

	// Set the service description.
	if err := s.UpdateConfig(mgr.Config{
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		StartType:   mgr.StartAutomatic,
		ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
	}); err != nil {
		// Non-fatal — description is cosmetic.
		fmt.Printf("[service] warning: could not set description: %v\n", err)
	}

	// Create the event log source.
	if err := eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		fmt.Printf("[service] warning: could not install event log source: %v\n", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("service: start service: %w", err)
	}

	fmt.Printf("[service] Service %q installed and started\n", serviceName)
	return nil
}

// uninstallService stops and removes the Windows service.
func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service: connect to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service: open service %q: %w", serviceName, err)
	}
	defer s.Close()

	// Stop the service and wait up to 10 seconds.
	status, err := s.Control(svc.Stop)
	if err == nil {
		deadline := time.Now().Add(10 * time.Second)
		for status.State != svc.Stopped && time.Now().Before(deadline) {
			time.Sleep(300 * time.Millisecond)
			status, err = s.Query()
			if err != nil {
				break
			}
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("service: delete service: %w", err)
	}

	if err := eventlog.Remove(serviceName); err != nil {
		fmt.Printf("[service] warning: could not remove event log source: %v\n", err)
	}

	fmt.Printf("[service] Service %q removed\n", serviceName)
	return nil
}

// serviceStatus returns a human-readable status string for the service.
func serviceStatus() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("service: connect to SCM: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		return "", fmt.Errorf("service: open service %q: %w", serviceName, err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("service: query service: %w", err)
	}

	switch status.State {
	case svc.Stopped:
		return "stopped", nil
	case svc.StartPending:
		return "start pending", nil
	case svc.StopPending:
		return "stop pending", nil
	case svc.Running:
		return "running", nil
	case svc.ContinuePending:
		return "continue pending", nil
	case svc.PausePending:
		return "pause pending", nil
	case svc.Paused:
		return "paused", nil
	default:
		return fmt.Sprintf("unknown (%d)", status.State), nil
	}
}