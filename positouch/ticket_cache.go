package positouch

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

// Ticket represents a POSitouch ticket/check for caching.
type Ticket struct {
	Number         int        `json:"number"`
	OpenedAt       time.Time  `json:"opened_at"`
	Table          int        `json:"table"`
	Total          float64    `json:"total"`
	ServerName     string     `json:"server_name"`
	PartySize      int        `json:"party_size"`
	CostCenter     int        `json:"cost_center"`
	CostCenterName string     `json:"cost_center_name"`
	Items          []Item     `json:"items"`
	Open           bool       `json:"open"`
}

// Item represents an item on a ticket/check.
type Item struct {
	ItemNumber         int     `json:"item_number"`
	ItemName           string  `json:"item_name"`
	CellName           string  `json:"cell_name"`
	MajorName          string  `json:"major_name"`
	MajorNumber        int     `json:"major_number"`
	MinorName          string  `json:"minor_name"`
	MinorNumber        int     `json:"minor_number"`
	ScreenNumber       int     `json:"screen_number"`
	CellNumber         int     `json:"cell_number"`
	FullPrice          float64 `json:"full_price"`
	NetPrice           float64 `json:"net_price"`
	PriceLevel         int     `json:"price_level"`
	SendTime           string  `json:"send_time"`
	SerialNumber       int     `json:"serial_number"`
	PrepSequence       int     `json:"prep_sequence"`
	PrepSequenceName   string  `json:"prep_sequence_name"`
	SplitItem          string  `json:"split_item"`
	SentToPrep         string  `json:"sent_to_prep"`
	NewServer          int     `json:"new_server"`
	NewServerName      string  `json:"new_server_name"`
	Option             *OptionDetail `json:"option,omitempty"`
}

// OptionDetail represents the nested <Option> in some ItemDetails.
type OptionDetail struct {
	ItemNumber   int    `xml:"ItemNumber" json:"item_number"`
	ItemName     string `xml:"ItemName" json:"item_name"`
	CellName     string `xml:"CellName" json:"cell_name"`
	Memo         string `xml:"Memo" json:"memo"`
	MajorName    string `xml:"MajorName" json:"major_name"`
	MajorNumber  int    `xml:"MajorNumber" json:"major_number"`
	MinorName    string `xml:"MinorName" json:"minor_name"`
	MinorNumber  int    `xml:"MinorNumber" json:"minor_number"`
	ScreenNumber int    `xml:"ScreenNumber" json:"screen_number"`
	CellNumber   int    `xml:"CellNumber" json:"cell_number"`
}

// XML Structs

type OpenChecks struct {
	XMLName xml.Name `xml:"OpenChecks"`
	Checks  []Check  `xml:"Check"`
}

type CheckFinalization struct {
	XMLName xml.Name `xml:"CheckFinalization"`
	Checks  []Check  `xml:"Check"`
}

type Check struct {
	Header CheckHeader   `xml:"CheckHeader"`
	Items  []ItemDetail  `xml:"ItemDetail"`
}

type CheckHeader struct {
	CheckNumber         int     `xml:"CheckNumber"`
	CheckOpenDate       string  `xml:"CheckOpenDate"`
	CheckOpenTime       string  `xml:"CheckOpenTime"`
	TableNumber         int     `xml:"TableNumber"`
	CheckTotal          float64 `xml:"CheckTotal"`
	ServerName          string  `xml:"ServerName"`
	NumberInParty       int     `xml:"NumberInParty"`
	CostCenter          int     `xml:"CostCenter"`
	CostCenterName      string  `xml:"CostCenterName"`
}

// ItemDetail represents each <ItemDetail>.
type ItemDetail struct {
	SplitItem         string        `xml:"SplitItem"`
	SentToPrep        string        `xml:"SentToPrep"`
	ItemNumber        int           `xml:"ItemNumber"`
	ItemName          string        `xml:"ItemName"`
	CellName          string        `xml:"CellName"`
	MajorName         string        `xml:"MajorName"`
	MajorNumber       int           `xml:"MajorNumber"`
	MinorName         string        `xml:"MinorName"`
	MinorNumber       int           `xml:"MinorNumber"`
	ScreenNumber      int           `xml:"ScreenNumber"`
	CellNumber        int           `xml:"CellNumber"`
	FullPrice         float64       `xml:"FullPrice"`
	NetPrice          float64       `xml:"NetPrice"`
	PriceLevel        int           `xml:"PriceLevel"`
	SendTime          string        `xml:"SendTime"`
	SerialNumber      int           `xml:"SerialNumber"`
	PrepSequence      int           `xml:"PrepSequence"`
	PrepSequenceName  string        `xml:"PrepSequenceName"`
	NewServer         int           `xml:"NewServer"`
	NewServerName     string        `xml:"NewServerName"`
	Option            *OptionDetail `xml:"Option"`
}

func parseTicketsFromXMLFiles(xmlDir string, open bool) ([]Ticket, error) {
	filesUpper, _ := filepath.Glob(filepath.Join(xmlDir, "*.XML"))
	filesLower, _ := filepath.Glob(filepath.Join(xmlDir, "*.xml"))
	files := append(filesUpper, filesLower...)

	// Only keep summary log for file count
	// log.Printf("[ticket_cache] XML files found: %+v", files)
	if len(files) == 0 {
		log.Printf("[ticket_cache] no XML files found in %s", xmlDir)
	}

	ticketMap := make(map[int]Ticket)

	for _, f := range files {
		xmlBytes, err := ioutil.ReadFile(f)
		if err != nil {
			log.Printf("[ticket_cache] unable to read file %s: %v", f, err)
			continue
		}

		// Determine root element and try both
		root := struct {
			XMLName xml.Name
		}{}
		if err := xml.Unmarshal(xmlBytes, &root); err != nil {
			log.Printf("[ticket_cache] unable to determine root of %s: %v", f, err)
			continue
		}

		var checks []Check
		switch root.XMLName.Local {
		case "OpenChecks":
			var openChecks OpenChecks
			if err := xml.Unmarshal(xmlBytes, &openChecks); err != nil {
				log.Printf("[ticket_cache] XML unmarshal error in %s (OpenChecks): %v", f, err)
				continue
			}
			checks = openChecks.Checks
		case "CheckFinalization":
			var cf CheckFinalization
			if err := xml.Unmarshal(xmlBytes, &cf); err != nil {
				log.Printf("[ticket_cache] XML unmarshal error in %s (CheckFinalization): %v", f, err)
				continue
			}
			checks = cf.Checks
		default:
			log.Printf("[ticket_cache] unknown XML root in %s: %s", f, root.XMLName.Local)
			continue
		}

		for _, chk := range checks {
			openedStr := chk.Header.CheckOpenDate + " " + chk.Header.CheckOpenTime
			openedAt, err := time.Parse("01/02/2006 15:04:05", openedStr)
			if err != nil {
				openedAt = time.Now()
			}
			ticket := Ticket{
				Number:         chk.Header.CheckNumber,
				OpenedAt:       openedAt,
				Table:          chk.Header.TableNumber,
				Total:          chk.Header.CheckTotal,
				ServerName:     chk.Header.ServerName,
				PartySize:      chk.Header.NumberInParty,
				CostCenter:     chk.Header.CostCenter,
				CostCenterName: chk.Header.CostCenterName,
				Open:           open,
			}
			for _, xmlItem := range chk.Items {
				item := Item{
					ItemNumber:       xmlItem.ItemNumber,
					ItemName:         xmlItem.ItemName,
					CellName:         xmlItem.CellName,
					MajorName:        xmlItem.MajorName,
					MajorNumber:      xmlItem.MajorNumber,
					MinorName:        xmlItem.MinorName,
					MinorNumber:      xmlItem.MinorNumber,
					ScreenNumber:     xmlItem.ScreenNumber,
					CellNumber:       xmlItem.CellNumber,
					FullPrice:        xmlItem.FullPrice,
					NetPrice:         xmlItem.NetPrice,
					PriceLevel:       xmlItem.PriceLevel,
					SendTime:         xmlItem.SendTime,
					SerialNumber:     xmlItem.SerialNumber,
					PrepSequence:     xmlItem.PrepSequence,
					PrepSequenceName: xmlItem.PrepSequenceName,
					SplitItem:        xmlItem.SplitItem,
					SentToPrep:       xmlItem.SentToPrep,
					NewServer:        xmlItem.NewServer,
					NewServerName:    xmlItem.NewServerName,
					Option:           xmlItem.Option,
				}
				ticket.Items = append(ticket.Items, item)
			}
			ticketMap[ticket.Number] = ticket // Deduplicate by CheckNumber
		}
	}

	var tickets []Ticket
	for _, t := range ticketMap {
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// ReadAllTickets returns all open and closed tickets, marked with their state.
func ReadAllTickets(openDir, closeDir string) ([]Ticket, error) {
	openTickets, _ := parseTicketsFromXMLFiles(openDir, true)
	closedTickets, _ := parseTicketsFromXMLFiles(closeDir, false)

	allTickets := make(map[int]Ticket)
	for _, t := range openTickets {
		allTickets[t.Number] = t
	}
	for _, t := range closedTickets {
		allTickets[t.Number] = t
	}

	result := []Ticket{}
	for _, t := range allTickets {
		result = append(result, t)
	}
	log.Printf("[ticket_cache] found %d unique tickets (open+closed)", len(result))
	return result, nil
}