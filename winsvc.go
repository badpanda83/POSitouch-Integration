//go:build windows

// winsvc.go — Windows Service Control Manager integration for the Rooam POS Agent.
//
// When the agent binary is launched by the Windows SCM (i.e. as a service),
// svc.IsWindowsService() returns true and we hand control to svc.Run(), which:
//
//  1. Calls agentService.Execute(), which reports StartPending → Running to SCM.
//  2. Blocks inside Execute() processing SCM commands until Stop or Shutdown.
//  3. Returns to runAsWindowsService, which returns true to main().
//  4. main() returns — the process exits cleanly.
//
// When run interactively (e.g. .\rooam-pos-agent.exe -config .\rooam_config.json),
// svc.IsWindowsService() returns false and runAsWindowsService returns false
// immediately, so main() continues with its normal startup and signal handling.

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

const windowsServiceName = "RooamPOSAgent"

// agentService implements svc.Handler so the Windows SCM can manage the agent.
type agentService struct{}

// Execute is called by svc.Run when the SCM starts the service.
// It must signal Running within the SCM timeout (~30 s) or the service fails to start.
func (s *agentService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	// Signal StartPending then Running as quickly as possible to satisfy the SCM
	// timeout (Error 7009). The agent's sync goroutines were already started by
	// main() before runAsWindowsService was called, so the service is functional.
	status <- svc.Status{State: svc.StartPending}
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}
	log.Printf("[winsvc] service %q is running", windowsServiceName)

	for c := range req {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			log.Printf("[winsvc] received SCM command %v — stopping", c.Cmd)
			status <- svc.Status{State: svc.StopPending}
			return false, 0
		case svc.Interrogate:
			status <- c.CurrentStatus
		default:
			log.Printf("[winsvc] unexpected SCM command %v — ignoring", c.Cmd)
		}
	}
	return false, 0
}

// runAsWindowsService checks whether we were launched by the SCM and, if so,
// calls svc.Run (which blocks until Stop/Shutdown) and returns true.
// The caller (main) should return immediately after this returns true.
// If running interactively, returns false without blocking.
func runAsWindowsService(configPath string) bool {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine if running as service: %v — assuming interactive", err)
		return false
	}
	if !isSvc {
		return false
	}
	if err := svc.Run(windowsServiceName, &agentService{}); err != nil {
		log.Printf("[winsvc] svc.Run returned error: %v", err)
	}
	return true
}