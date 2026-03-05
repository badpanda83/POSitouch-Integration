// Package positouchdriver implements the driver.POSDriver interface for POSitouch.
package positouchdriver

import (
	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// Driver wraps POSitouch-specific packages and satisfies the driver.POSDriver interface.
type Driver struct {
	cfg *config.Config
}

// New creates a new POSitouch Driver from the given configuration.
func New(cfg *config.Config) *Driver {
	return &Driver{cfg: cfg}
}

// Name returns the identifier for this driver.
func (d *Driver) Name() string { return "positouch" }

// Ensure *Driver satisfies the POSDriver interface at compile time.
// (The three method bodies are in sync.go, tickets.go and ordering.go.)
var _ interface {
	SyncEntities() (*entities.Snapshot, error)
	SyncTickets() ([]entities.Ticket, error)
	CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error)
	Name() string
} = (*Driver)(nil)
