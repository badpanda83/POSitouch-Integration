// Package micros3700driver implements the driver.POSDriver interface for MICROS RES 3700.
// Tickets are received via the IFS RTTP TCP push interface (port 5454).
// Master data (entities) is read from Sybase SQL Anywhere 16 via ODBC on Windows.
package micros3700driver

import (
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// Driver implements driver.POSDriver for MICROS RES 3700.
type Driver struct {
	cfg      *config.Config
	listener *RttpListener
}

// New creates a new MICROS 3700 Driver, starts the RTTP TCP listener, and
// returns the driver ready for use.
func New(cfg *config.Config) *Driver {
	port := 5454
	if cfg.MICROS3700 != nil && cfg.MICROS3700.RttpPort > 0 {
		port = cfg.MICROS3700.RttpPort
	}
	d := &Driver{
		cfg:      cfg,
		listener: NewRttpListener(port),
	}
	go d.listener.Start()
	return d
}

// Name returns the identifier for this driver.
func (d *Driver) Name() string { return "micros3700" }

// SyncTickets returns all open tickets currently held in the RTTP in-memory store.
func (d *Driver) SyncTickets() ([]entities.Ticket, error) {
	return d.listener.Tickets(), nil
}

// SyncEntities reads all master data (employees, tables, tenders, cost centers,
// order types, menu items) from the Sybase SQL Anywhere 16 database via ODBC.
// On non-Windows builds or when ODBC is unavailable, an empty snapshot is returned.
func (d *Driver) SyncEntities() (*entities.Snapshot, error) {
	return syncEntitiesODBC(d.cfg)
}

// Ensure *Driver satisfies the POSDriver interface at compile time.
var _ interface {
	SyncEntities() (*entities.Snapshot, error)
	SyncTickets() ([]entities.Ticket, error)
	CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error)
	Name() string
} = (*Driver)(nil)
