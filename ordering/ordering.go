package ordering

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// --- API request/response models ---

type CreateTicketRequest struct {
	ReferenceNumber string             `json:"reference_number"`
	TableNumber     string             `json:"table_number"`
	CostCenter      string             `json:"cost_center,omitempty"`
	ServerNumber    string             `json:"server_number"`
	TerminalNumber  string             `json:"terminal_number,omitempty"`
	NumberInParty   int                `json:"number_in_party,omitempty"`
	Items           []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	ItemNumber   string            `json:"item_number"`
	ScreenCell   string            `json:"screen_cell,omitempty"`
	ItemName     string            `json:"item_name,omitempty"`
	Quantity     int               `json:"quantity"`
	CategoryID   string            `json:"category_id,omitempty"`
	Memo         string            `json:"memo,omitempty"`
	Modifiers    []ModifierRequest `json:"modifiers,omitempty"`
}

type ModifierRequest struct {
	ItemNumber string `json:"item_number"`
	ScreenCell string `json:"screen_cell,omitempty"`
	ItemName   string `json:"item_name,omitempty"`
	Quantity   int    `json:"quantity,omitempty"`
	Memo       string `json:"memo,omitempty"`
}

// --- API response model ---
type CreateTicketResponse struct {
	Status          string `json:"status"`
	ReferenceNumber string `json:"reference_number,omitempty"`
	Message         string `json:"message,omitempty"`
}

// --- POSitouch Ordering XML models ---

type Orders struct {
	XMLName  xml.Name   `xml:"Orders"`
	NewOrder *NewOrder  `xml:"NewOrder,omitempty"`
}

type NewOrder struct {
	Function        int    `xml:"Function"`      // 1 == Open Check/Order
	ErrorLevel      int    `xml:"ErrorLevel"`
	ReferenceNumber string `xml:"ReferenceNumber,omitempty"`
	Check           *Check `xml:"Check,omitempty"`
}

type Check struct {
	CheckHeader CheckHeader   `xml:"CheckHeader"`
	ItemDetails []ItemDetail  `xml:"ItemDetail,omitempty"`
}

type CheckHeader struct {
	TableNumber    string `xml:"TableNumber"`
	CostCenter     string `xml:"CostCenter,omitempty"`
	ServerNumber   string `xml:"ServerNumber"`
	NumberInParty  int    `xml:"NumberInParty,omitempty"`
	TerminalNumber string `xml:"TerminalNumber,omitempty"`
}

type ItemDetail struct {
	ItemNumber string   `xml:"ItemNumber,omitempty"`
	ScreenCell string   `xml:"ScreenCell,omitempty"`
	ItemName   string   `xml:"ItemName,omitempty"`
	Quantity   int      `xml:"Quantity,omitempty"`
	Memo       string   `xml:"Memo,omitempty"`
	CategoryID string   `xml:"CategoryID,omitempty"`
	Options    []Option `xml:"Option,omitempty"`
}

type Option struct {
	ItemNumber string `xml:"ItemNumber,omitempty"`
	ScreenCell string `xml:"ScreenCell,omitempty"`
	ItemName   string `xml:"ItemName,omitempty"`
	Quantity   int    `xml:"Quantity,omitempty"`
	Memo       string `xml:"Memo,omitempty"`
}

// --- Main API handler ---

// POST /api/v1/pos-data/{locationId}/tickets
func CreateTicket(w http.ResponseWriter, r *http.Request, locationID string) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	checkHeader := CheckHeader{
		TableNumber:    req.TableNumber,
		CostCenter:     req.CostCenter,
		ServerNumber:   req.ServerNumber,
		NumberInParty:  req.NumberInParty,
		TerminalNumber: req.TerminalNumber,
	}

	order := Orders{
		NewOrder: &NewOrder{
			Function:        1,
			ErrorLevel:      2,
			ReferenceNumber: req.ReferenceNumber,
			Check: &Check{
				CheckHeader: checkHeader,
			},
		},
	}

	for _, it := range req.Items {
		item := ItemDetail{
			ItemNumber: it.ItemNumber,
			ScreenCell: it.ScreenCell,
			ItemName:   it.ItemName,
			Quantity:   it.Quantity,
			Memo:       it.Memo,
			CategoryID: it.CategoryID,
		}
		for _, mod := range it.Modifiers {
			item.Options = append(item.Options, Option{
				ItemNumber: mod.ItemNumber,
				ScreenCell: mod.ScreenCell,
				ItemName:   mod.ItemName,
				Quantity:   mod.Quantity,
				Memo:       mod.Memo,
			})
		}
		order.NewOrder.Check.ItemDetails = append(order.NewOrder.Check.ItemDetails, item)
	}

	xmlData, err := xml.MarshalIndent(order, "", "  ")
	if err != nil {
		http.Error(w, "failed to marshal xml", http.StatusInternalServerError)
		return
	}

	outDir := "/SC/ORDERS" // TODO: set to your actual ingest directory!
	if err := writeOrderXMLAtomically(xmlData, outDir); err != nil {
		http.Error(w, "failed to write order file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(CreateTicketResponse{
		Status:          "accepted",
		ReferenceNumber: req.ReferenceNumber,
	})
}

func AddTicketItems(w http.ResponseWriter, r *http.Request, locationID, ticketID string) {
	http.Error(w, "AddTicketItems not yet implemented", http.StatusNotImplemented)
}

func AddTicketPayments(w http.ResponseWriter, r *http.Request, locationID, ticketID string) {
	http.Error(w, "AddTicketPayments not yet implemented", http.StatusNotImplemented)
}

func writeOrderXMLAtomically(xmlData []byte, dir string) error {
	tmp := filepath.Join(dir, fmt.Sprintf("ORDER_%d.tmp", time.Now().UnixNano()))
	final := strings.TrimSuffix(tmp, ".tmp") + ".XML"
	if err := os.WriteFile(tmp, xmlData, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}