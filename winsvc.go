//go:build windows

// winsvc.go — Windows Service Control Manager integration for the Rooam POS Agent.
//
// When the agent binary is launched by the Windows SCM (i.e. as a service),
// svc.IsWindowsService() returns true and we hand control to svc.Run(), which:
//
//  1. Calls agentService.Execute(), which reports StartPending → Running to SCM.
//  2. Runs the agent loop (runAgent) in the background.
//  3. Blocks until SCM sends a Stop or Shutdown command, then signals runAgent to exit.
//
// When run interactively (e.g. .\rooam-pos-agent.exe -config .\rooam_config.json),
// svc.IsWindowsService() returns false and we fall straight through to runAgent,
// which uses os/signal (SIGINT / SIGTERM) as before — no behaviour change.

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"
)

const windowsServiceName = "RooamPOSAgent"

// agentService implements svc.Handler so the Windows SCM can manage the agent.
type agentService struct {
	configPath string
}

// Execute is called by svc.Run when the SCM starts the service.
// It must signal Running within the SCM timeout (~30 s) or the service fails to start.
func (s *agentService) Execute(args []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	// 1. Acknowledge that we are starting.
	status <- svc.Status{State: svc.StartPending}

	// 2. Start the agent run loop in a goroutine so we can return Running immediately.
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
go func() {
		defer close(doneCh)
		runAgent(s.configPath, stopCh)
	}()

	// 3. Tell SCM we are running — this is the call that fixes the 7009 timeout.
	status <- svc.Status{
		State:   svc.Running,
		Accepts: svc.AcceptStop | svc.AcceptShutdown,
	}
	log.Printf("[winsvc] service %q is running", windowsServiceName)

	// 4. Process SCM control requests.
	for {
		select {
		case c := <-req:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				log.Printf("[winsvc] received SCM command %v — stopping", c.Cmd)
				status <- svc.Status{State: svc.StopPending}
				close(stopCh) // signal runAgent to exit
				<-doneCh      // wait for clean shutdown
				return false, 0
			case svc.Interrogate:
				status <- c.CurrentStatus
			default:
				log.Printf("[winsvc] unexpected SCM command %v — ignoring", c.Cmd)
			}
		case <-doneCh:
			// runAgent returned on its own (should not normally happen).
			log.Printf("[winsvc] agent loop exited unexpectedly — stopping service")
			return false, 0
		}
	}
}

// runAsWindowsService checks whether we were launched by the SCM and, if so,
// hands control to svc.Run and returns true. The caller should return immediately.
// If we are running interactively it returns false.
func runAsWindowsService(configPath string) bool {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Printf("[winsvc] could not determine if running as service: %v — assuming interactive", err)
		return false
	}
	if !isSvc {
		return false
	}
	if err := svc.Run(windowsServiceName, &agentService{configPath: configPath}); err != nil {
		log.Printf("[winsvc] svc.Run returned error: %v", err)
	}
	return true
}