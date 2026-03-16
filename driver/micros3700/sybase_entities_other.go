//go:build !windows || !cgo

package micros3700driver

import (
	"log"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// syncEntitiesODBC is a stub for non-Windows or CGO-disabled builds.
// ODBC access to Sybase SQL Anywhere 16 requires Windows and CGO.
// Tickets (via the RTTP TCP listener) continue to work without ODBC.
func syncEntitiesODBC(cfg *config.Config) (*entities.Snapshot, error) {
	log.Printf("[micros3700][WARN] SyncEntities: ODBC is only available on Windows (CGO build); returning empty snapshot")
	return &entities.Snapshot{}, nil
}
