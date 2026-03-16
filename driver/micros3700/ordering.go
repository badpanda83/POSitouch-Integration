package micros3700driver

import (
	"fmt"

	"github.com/badpanda83/POSitouch-Integration/entities"
)

// CreateOrder is not supported for MICROS 3700 via ODBC. Order creation requires
// direct access to the POS terminal and is not available through the database interface.
func (d *Driver) CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error) {
	return nil, fmt.Errorf("micros3700: CreateOrder is not supported via ODBC; use the POS terminal directly")
}
