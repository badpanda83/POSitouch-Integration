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

// quotePath wraps a path in double-quotes so that the Windows SCM handles
// paths that contain spaces correctly.  The SCM uses the raw ImagePath value
// as a command-line, so each token (exe and arguments) must be individually
// quoted when it may contain spaces.
func quotePath(p string) string {
	return `"` + p + `"`
}

// installService creates and starts the Windows service, then polls until it
// reaches svc.Running (or times out after 30 seconds).
func installService(agentPath, configPath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("service: connect to SCM: %w", err)
	}
	defer m.Disconnect()

	// Quote both the exe path and the config path argument so that paths
	// containing spaces are handled correctly by the SCM.
	binaryPathName := quotePath(agentPath) + " -config " + quotePath(configPath)

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

	// Ask the SCM to start the service.
	if err := s.Start(); err != nil {
		return fmt.Errorf("service: start service: %w", err)
	}

	// Poll until the service reaches Running or we time out.
	fmt.Printf("[service] Waiting for service %q to start", serviceName)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("service: query status after start: %w", err)
		}
		if status.State == svc.Running {
			fmt.Println() // newline after the dots
			fmt.Printf("[service] Service %q is running\n", serviceName)
			return nil
		}
		if status.State != svc.StartPending {
			fmt.Println()
			return fmt.Errorf("service: unexpected state after start: %d", status.State)
		}
		fmt.Print(".")
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println()
	return fmt.Errorf("service: timed out waiting for %q to reach Running state", serviceName)
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
		return "not installed", nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("service: query status: %w", err)
	}

	switch status.State {
	case svc.Running:
		return "running", nil
	case svc.Stopped:
		return "stopped", nil
	case svc.StartPending:
		return "start pending", nil
	case svc.StopPending:
		return "stop pending", nil
	case svc.Paused:
		return "paused", nil
	case svc.PausePending:
		return "pause pending", nil
	case svc.ContinuePending:
		return "continue pending", nil
	default:
		return fmt.Sprintf("unknown (%d)", status.State), nil
	}
}