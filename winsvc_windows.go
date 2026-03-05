//go:build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

// posAgent implements svc.Handler so the Windows SCM can control the agent.
type posAgent struct {
	stop chan struct{}
}

// Execute is called by the SCM in a dedicated goroutine.
func (a *posAgent) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

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

// runAsWindowsService detects whether the process was started by the Windows
// SCM. If so it starts the dispatch loop and returns (true, stopCh). When run
// interactively it returns (false, nil).
func runAsWindowsService(serviceName string) (bool, <-chan struct{}) {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine service context: %v — assuming interactive", err)
		return false, nil
	}
	if !isSvc {
		return false, nil
	}

	stop := make(chan struct{})
	go func() {
		if err := svc.Run(serviceName, &posAgent{stop: stop}); err != nil {
			log.Printf("[winsvc] svc.Run returned: %v", err)
		}
	}() 
	return true, stop
}