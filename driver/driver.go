// Package driver defines the POSDriver interface that every POS integration must satisfy.
package driver

import "github.com/badpanda83/POSitouch-Integration/entities"

// POSDriver is the contract every POS integration must satisfy.
type POSDriver interface {
	// SyncEntities reads all master data (tables, employees, tenders, etc.) from
	// the POS system and returns a canonical Snapshot.
	SyncEntities() (*entities.Snapshot, error)

	// SyncTickets reads all open and closed tickets from the POS system and
	// returns them as canonical Ticket values.
	SyncTickets() ([]entities.Ticket, error)

	// CreateOrder places a new order on the POS system and returns the resulting
	// ticket once the POS confirms it.
	CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error)

	// Name returns a human-readable identifier for this driver (e.g. "positouch").
	Name() string
}
