//go:build !windows || !cgo

package micros3700driver

import (
	"log"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// syncEntitiesODBC is a stub for non-Windows or CGO-disabled builds.
// ODBC access to Sybase SQL Anywhere 16 requires Windows and CGO.
func syncEntitiesODBC(cfg *config.Config) (*entities.Snapshot, error) {
	log.Printf("[micros3700][WARN] ODBC entity sync is only supported on Windows with CGo enabled; returning empty snapshot")
	return &entities.Snapshot{}, nil
}

// syncTicketsODBC is a stub for non-Windows or CGO-disabled builds.
func syncTicketsODBC(cfg *config.Config) ([]entities.Ticket, error) {
	log.Printf("[micros3700][WARN] ODBC ticket sync is only supported on Windows with CGo enabled; returning empty ticket list")
	return []entities.Ticket{}, nil
}
