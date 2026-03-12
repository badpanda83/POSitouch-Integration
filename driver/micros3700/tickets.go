package micros3700driver

import (
	"encoding/xml"
	"log"

	"github.com/badpanda83/POSitouch-Integration/entities"
)

// --- XML request/response types ---

type getOpenChecksRequest struct {
	XMLName                xml.Name `xml:"http://www.micros.com/res/pos/webservices/general/v1 GetOpenChecks"`
	RevenueCenterObjectNum int      `xml:"RevenueCenterObjectNum"`
}

type getOpenChecksResponse struct {
	XMLName xml.Name      `xml:"GetOpenChecksResponse"`
	Checks  []microsCheck `xml:"Checks>Check"`
}

type microsCheck struct {
	CheckNum               int               `xml:"CheckNum"`
	TableObjectNum         int               `xml:"TableObjectNum"`
	EmployeeName           string            `xml:"EmployeeName"`
	NumberInParty          int               `xml:"NumberInParty"`
	CheckTotal             float64           `xml:"CheckTotal"`
	OpenTime               string            `xml:"OpenTime"`
	RevenueCenterObjectNum int               `xml:"RevenueCenterObjectNum"`
	RevenueCenterName      string            `xml:"RevenueCenterName"`
	CheckItems             []microsCheckItem `xml:"CheckItems>CheckItem"`
}

type microsCheckItem struct {
	MenuItemObjectNum int              `xml:"MenuItemObjectNum"`
	MenuItemName      string           `xml:"MenuItemName"`
	Quantity          int              `xml:"Quantity"`
	Price             float64          `xml:"Price"`
	Modifiers         []microsModifier `xml:"Modifiers>Modifier"`
}

type microsModifier struct {
	MenuItemObjectNum int    `xml:"MenuItemObjectNum"`
	MenuItemName      string `xml:"MenuItemName"`
	Quantity          int    `xml:"Quantity"`
}

// SyncTickets calls the MICROS 3700 Transaction Services endpoint to retrieve all
// open checks and maps them to canonical entities.Ticket values.
func (d *Driver) SyncTickets() ([]entities.Ticket, error) {
	mcfg := d.cfg.MICROS3700
	if mcfg == nil {
		log.Printf("[micros3700][WARN] micros3700 config is nil; returning empty ticket list")
		return []entities.Ticket{}, nil
	}

	reqPayload := getOpenChecksRequest{
		RevenueCenterObjectNum: mcfg.RevenueCenterID,
	}

	respBody, err := postSOAP(mcfg, microsNS+"/GetOpenChecks", reqPayload)
	if err != nil {
		log.Printf("[micros3700][WARN] SyncTickets: %v", err)
		return []entities.Ticket{}, nil
	}

	var parsed getOpenChecksResponse
	if err := xml.Unmarshal(respBody, &parsed); err != nil {
		log.Printf("[micros3700][WARN] SyncTickets: unmarshal response: %v", err)
		return []entities.Ticket{}, nil
	}

	tickets := make([]entities.Ticket, 0, len(parsed.Checks))
	for _, c := range parsed.Checks {
		tickets = append(tickets, mapMicrosCheck(c))
	}
	return tickets, nil
}

func mapMicrosCheck(c microsCheck) entities.Ticket {
	items := make([]entities.TicketItem, 0, len(c.CheckItems))
	for _, it := range c.CheckItems {
		items = append(items, mapMicrosCheckItem(it))
	}
	return entities.Ticket{
		Number:         c.CheckNum,
		OpenedAt:       c.OpenTime,
		Table:          c.TableObjectNum,
		Total:          c.CheckTotal,
		ServerName:     c.EmployeeName,
		PartySize:      c.NumberInParty,
		CostCenter:     c.RevenueCenterObjectNum,
		CostCenterName: c.RevenueCenterName,
		Items:          items,
		Open:           true,
		POSType:        "micros3700",
	}
}

func mapMicrosCheckItem(it microsCheckItem) entities.TicketItem {
	return entities.TicketItem{
		ItemNumber: it.MenuItemObjectNum,
		ItemName:   it.MenuItemName,
		FullPrice:  it.Price,
		NetPrice:   it.Price,
	}
}
