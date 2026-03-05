//go:build windows

// winsvc_windows.go — Windows Service Control Manager integration for the Rooam POS Agent.
//
// When the agent binary is launched by the Windows SCM (i.e. as a service),
// svc.IsWindowsService() returns true. runAsWindowsService starts the SCM
// dispatch loop in a goroutine and returns (true, stopCh). main() continues
// with all agent logic unconditionally and uses stopCh only for graceful
// shutdown alongside OS signals.
//
// When run interactively, svc.IsWindowsService() returns false and
// runAsWindowsService returns (false, nil) immediately — no behaviour change.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
)

// redirectLogToFile redirects the standard logger to a date-stamped log file
// next to the running executable. This is called only when running as a
// Windows Service, because services have no console stdout/stderr.
func redirectLogToFile() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	name := fmt.Sprintf("rooam-pos-agent-%s.log", time.Now().Format("2006-01-02"))
	path := filepath.Join(dir, name)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.LUTC)
	log.Printf("[winsvc] logging to %s", path)
}

// posAgent implements svc.Handler so the Windows SCM can control the agent.
type posAgent struct {
	stop chan struct{}
}

// Execute is called by the SCM in a dedicated goroutine.
// It signals StartPending → Running as quickly as possible to avoid the
// 7009 "service did not respond in time" error, then waits for Stop/Shutdown.
func (a *posAgent) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[winsvc] panic in Execute: %v — stopping service", rec)
			s <- svc.Status{State: svc.StopPending}
			close(a.stop)
		}
	}()

	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	log.Printf("[winsvc] service is running")

	for c := range r {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			log.Printf("[winsvc] received SCM command %v — stopping", c.Cmd)
			s <- svc.Status{State: svc.StopPending}
			close(a.stop)
			return false, 0
		case svc.Interrogate:
			s <- c.CurrentStatus
		default:
			log.Printf("[winsvc] unexpected SCM command %v — ignoring", c.Cmd)
		}
	}
	return false, 0
}

// runAsWindowsService detects whether the process was started by the Windows
// SCM. If so it redirects logging to a file, starts the dispatch loop and
// returns (true, stopCh). stopCh is closed when the SCM sends Stop or
// Shutdown. When run interactively it returns (false, nil).
func runAsWindowsService(serviceName string) (bool, <-chan struct{}) {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine service context: %v — assuming interactive", err)
		return false, nil
	}
	if !isSvc {
		return false, nil
	}

	// Redirect logs to a file before anything else so startup errors are visible.
	redirectLogToFile()

	stop := make(chan struct{})
	go func() {
		if err := svc.Run(serviceName, &posAgent{stop: stop}); err != nil {
			log.Printf("[winsvc] svc.Run returned: %v", err)
		}
	}()
	return true, stop
}