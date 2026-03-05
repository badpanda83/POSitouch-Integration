//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

// posAgent implements the windows/svc.Handler interface so that the Windows
// SCM can start, stop, and query the Rooam POS Agent service.
type posAgent struct {
	stop chan struct{}
}

// Execute is called by the Windows SCM in a dedicated goroutine.  It must
// accept the service start request, process SCM commands (at minimum Stop and
// Shutdown), and return when the service is done.
func (a *posAgent) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	// Report "Start Pending"
	s <- svc.Status{State: svc.StartPending}

	// Report "Running" — the actual worker goroutines are already running in
	// main() before runAsWindowsService is called.
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for c := range r {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending}
			close(a.stop)
			return false, 0
		default:
			log.Printf("[winsvc] unexpected SCM command: %v", c.Cmd)
		}
	}
	return false, 0
}

// runAsWindowsService detects whether the process was launched by the Windows
// SCM (rather than interactively from a console).  If it was, it starts the
// SCM dispatch loop in the current goroutine and returns (true, stopCh) where
// stopCh is closed when the SCM sends a Stop/Shutdown command.  If the process
// is running interactively it returns (false, nil) immediately.
func runAsWindowsService(serviceName string) (bool, <-chan struct{}) {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine service context: %v — assuming interactive", err)
		return false, nil
	}
	if !isService {
		return false, nil
	}

	stop := make(chan struct{})
	agent := &posAgent{stop: stop}

	go func() {
		if err := svc.Run(serviceName, agent); err != nil {
			log.Printf("[winsvc] svc.Run returned: %v", err)
		}
	}()

	return true, stop
}