// Package micros3700driver implements the driver.POSDriver interface for MICROS 3700.
package micros3700driver

import (
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// Driver implements driver.POSDriver for MICROS 3700.
type Driver struct {
	cfg *config.Config
}

// New creates a new MICROS 3700 Driver from the given configuration.
func New(cfg *config.Config) *Driver {
	return &Driver{cfg: cfg}
}

// Name returns the identifier for this driver.
func (d *Driver) Name() string { return "micros3700" }

// Ensure *Driver satisfies the POSDriver interface at compile time.
var _ interface {
	SyncEntities() (*entities.Snapshot, error)
	SyncTickets() ([]entities.Ticket, error)
	CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error)
	Name() string
} = (*Driver)(nil)
