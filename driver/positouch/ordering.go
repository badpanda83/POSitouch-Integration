package positouchdriver

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/badpanda83/POSitouch-Integration/entities"
	"github.com/badpanda83/POSitouch-Integration/ordering"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	orderConfirmTimeout    = 30 * time.Second
	orderConfirmInterval   = 2 * time.Second
	orderTicketMatchWindow = 60 * time.Second
)

// CreateOrder converts the canonical request to a POSitouch XML order, writes it to
// the inbound order directory, polls for confirmation, and returns the matched ticket.
func (d *Driver) CreateOrder(req entities.CreateOrderRequest) (*entities.Ticket, error) {
	ticketReq := toOrderingRequest(req)

	if err := ordering.WriteOrderXML(ticketReq, d.cfg.XMLInOrderDir); err != nil {
		return nil, fmt.Errorf("positouch: write order XML: %w", err)
	}

	deadline := time.Now().Add(orderConfirmTimeout)
	for time.Now().Before(deadline) {
		conf, confFile, err := ordering.FindConfirmation(d.cfg.XMLDir, req.ReferenceNumber)
		if err == nil && conf != nil {
			// Clean up the confirmation file.
			if removeErr := os.Remove(confFile); removeErr != nil {
				log.Printf("[positouch][WARN] failed to remove confirmation file %s: %v", confFile, removeErr)
			}

			if conf.Transaction.ResponseCode != 0 {
				errText := ""
				if conf.Transaction.Error != nil {
					errText = conf.Transaction.Error.Text
				}
				return nil, fmt.Errorf("positouch: order rejected (code %d): %s", conf.Transaction.ResponseCode, errText)
			}

			// Success — find the newly created ticket by table and time window.
			ticket, matchErr := d.findMatchedTicket(req.TableNumber)
			if matchErr != nil {
				log.Printf("[positouch][WARN] could not find matched ticket: %v", matchErr)
			}
			return ticket, nil
		}
		time.Sleep(orderConfirmInterval)
	}

	return nil, fmt.Errorf("positouch: timeout after %.0fs waiting for confirmation of order %s",
		orderConfirmTimeout.Seconds(), req.ReferenceNumber)
}

// toOrderingRequest converts a canonical CreateOrderRequest into the POSitouch-specific type.
func toOrderingRequest(req entities.CreateOrderRequest) ordering.CreateTicketRequest {
	items := make([]ordering.OrderItemRequest, 0, len(req.Items))
	for _, it := range req.Items {
		mods := make([]ordering.ModifierRequest, 0, len(it.Modifiers))
		for _, m := range it.Modifiers {
			mods = append(mods, ordering.ModifierRequest{
				ItemNumber: m.ItemNumber,
				ScreenCell: m.ScreenCell,
				ItemName:   m.ItemName,
				Quantity:   m.Quantity,
				Memo:       m.Memo,
			})
		}
		items = append(items, ordering.OrderItemRequest{
			ItemNumber:   it.ItemNumber,
			ScreenCell:   it.ScreenCell,
			ItemName:     it.ItemName,
			Quantity:     it.Quantity,
			CategoryID:   it.CategoryID,
			CategoryName: it.CategoryName,
			Memo:         it.Memo,
			Modifiers:    mods,
		})
	}
	return ordering.CreateTicketRequest{
		ReferenceNumber: req.ReferenceNumber,
		TableNumber:     req.TableNumber,
		CostCenter:      req.CostCenter,
		ServerNumber:    req.ServerNumber,
		TerminalNumber:  req.TerminalNumber,
		NumberInParty:   req.NumberInParty,
		Items:           items,
	}
}

// findMatchedTicket reads current tickets and returns the one matching tableNumber
// that was opened within the ticket-match window.
func (d *Driver) findMatchedTicket(tableNumber string) (*entities.Ticket, error) {
	tableNum, err := strconv.Atoi(tableNumber)
	if err != nil {
		return nil, fmt.Errorf("table_number %q is not a valid integer: %w", tableNumber, err)
	}

	tickets, err := positouch.ReadAllTickets(d.cfg.XMLDir, d.cfg.XMLCloseDir)
	if err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}

	for i := range tickets {
		t := &tickets[i]
		if t.Table == tableNum && time.Since(t.OpenedAt) <= orderTicketMatchWindow {
			mapped := mapTicket(*t)
			return &mapped, nil
		}
	}
	return nil, nil
}
