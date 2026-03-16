package micros3700driver

import (
	"fmt"

	"github.com/badpanda83/POSitouch-Integration/entities"
)

// CreateOrder is not supported by the MICROS 3700 RTTP driver.
// Order creation via MICROS 3700 requires a separate integration path
// not covered by the IFS push interface.
func (d *Driver) CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error) {
	return nil, fmt.Errorf("micros3700: CreateOrder is not supported by the RTTP driver")
}
