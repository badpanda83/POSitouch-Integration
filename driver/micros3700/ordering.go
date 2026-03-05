package micros3700driver

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/badpanda83/POSitouch-Integration/entities"
)

// --- XML request/response types for order creation ---

type checkAddRequest struct {
	XMLName xml.Name    `xml:"Checks"`
	Check   microsOrder `xml:"Check"`
}

type microsOrder struct {
	EmployeeObjectNum      int               `xml:"EmployeeObjectNum"`
	TableObjectNum         int               `xml:"TableObjectNum"`
	RevenueCenterObjectNum int               `xml:"RevenueCenterObjectNum"`
	CheckItems             []microsOrderItem `xml:"CheckItems>CheckItem"`
}

type microsOrderItem struct {
	MenuItemObjectNum int `xml:"MenuItemObjectNum"`
	Quantity          int `xml:"Quantity"`
}

type checkAddResponse struct {
	XMLName xml.Name         `xml:"CheckAddResponse"`
	Check   checkAddRespItem `xml:"Check"`
}

type checkAddRespItem struct {
	CheckNum int    `xml:"CheckNum"`
	Status   string `xml:"Status"`
	Error    string `xml:"Error"`
}

// CreateOrder converts the canonical request to a MICROS 3700 XML order, posts it
// to Transaction Services, and returns the resulting ticket.
func (d *Driver) CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error) {
	mcfg := d.cfg.MICROS3700
	if mcfg == nil {
		return nil, fmt.Errorf("micros3700: driver config is nil")
	}

	employeeNum, err := strconv.Atoi(req.ServerNumber)
	if err != nil {
		return nil, fmt.Errorf("micros3700: server_number %q is not a valid integer: %w", req.ServerNumber, err)
	}
	tableNum, err := strconv.Atoi(req.TableNumber)
	if err != nil {
		return nil, fmt.Errorf("micros3700: table_number %q is not a valid integer: %w", req.TableNumber, err)
	}

	rvcID := mcfg.RevenueCenterID
	if req.CostCenter != "" {
		if v, err := strconv.Atoi(req.CostCenter); err == nil {
			rvcID = v
		}
	}

	items := make([]microsOrderItem, 0, len(req.Items))
	for _, it := range req.Items {
		itemNum, err := strconv.Atoi(it.ItemNumber)
		if err != nil {
			return nil, fmt.Errorf("micros3700: item_number %q is not a valid integer: %w", it.ItemNumber, err)
		}
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		items = append(items, microsOrderItem{
			MenuItemObjectNum: itemNum,
			Quantity:          qty,
		})
	}

	payload := checkAddRequest{
		Check: microsOrder{
			EmployeeObjectNum:      employeeNum,
			TableObjectNum:         tableNum,
			RevenueCenterObjectNum: rvcID,
			CheckItems:             items,
		},
	}

	respBody, err := postXML(mcfg, payload)
	if err != nil {
		return nil, fmt.Errorf("micros3700: Transaction Services unreachable: %w", err)
	}

	var parsed checkAddResponse
	if err := xml.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("micros3700: unmarshal order response: %w", err)
	}

	if parsed.Check.Status != "Success" {
		return nil, fmt.Errorf("micros3700: order failed: %s", parsed.Check.Error)
	}

	// Find the newly created ticket by check number.
	tickets, err := d.SyncTickets()
	if err != nil {
		return nil, fmt.Errorf("micros3700: SyncTickets after order: %w", err)
	}
	for i := range tickets {
		if tickets[i].Number == parsed.Check.CheckNum {
			return &tickets[i], nil
		}
	}

	// Ticket created but not yet readable — not an error.
	return nil, nil
}
