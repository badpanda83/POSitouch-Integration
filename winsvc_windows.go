//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

// agentService implements svc.Handler so the agent can run under the Windows SCM.
type agentService struct {
	stop chan struct{} // closed when the SCM sends a Stop/Shutdown command
}

// Execute is called by the Windows SCM in a dedicated goroutine.
// It signals the main goroutine to shut down via the stop channel and waits
// for the agent to finish before returning.
func (a *agentService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	// Report that we are starting up.
	status <- svc.Status{State: svc.StartPending}

	// Signal the main goroutine that we are ready.
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}

	for c := range req {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.StopPending}
			close(a.stop)
			return false, 0
		default:
			log.Printf("[winsvc] unexpected SCM command: %d", c.Cmd)
		}
	}
	return false, 0
}

// runAsWindowsService detects whether the process is running as a Windows
// service and, if so, hands control to the SCM via svc.Run.  It returns true
// if it ran as a service (in which case main should return), and false if the
// process was started interactively.
//
// The returned channel is closed when the SCM sends a Stop/Shutdown signal;
// the caller should treat this as equivalent to SIGTERM.
func runAsWindowsService(serviceName string) (ranAsSvc bool, stop <-chan struct{}) {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine if running as service: %v", err)
		return false, nil
	}
	if !isSvc {
		return false, nil
	}

	stopCh := make(chan struct{})
	handler := &agentService{stop: stopCh}

	go func() {
		if err := svc.Run(serviceName, handler); err != nil {
			log.Printf("[winsvc] svc.Run returned: %v", err)
		}
	}()

	return true, stopCh
}