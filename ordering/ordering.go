package ordering

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	confirmPollInterval = 2 * time.Second
	confirmPollTimeout  = 30 * time.Second
	ticketMatchWindow   = 60 * time.Second
)

// --- Expanded Models for Ordering, Menu Items, Modifiers, Categories ---

type Category struct {
	ID          string     `json:"id" xml:"ID"`
	Name        string     `json:"name" xml:"Name"`
	Description string     `json:"description,omitempty" xml:"Description,omitempty"`
	Major       int        `json:"major,omitempty" xml:"Major,omitempty"`
	Minor       int        `json:"minor,omitempty" xml:"Minor,omitempty"`
	Minors      []Category `json:"minors,omitempty" xml:"Minors>Category,omitempty"`
}

type Modifier struct {
	ID          int     `json:"id" xml:"ID"`
	Name        string  `json:"name" xml:"Name"`
	Description string  `json:"description,omitempty" xml:"Description,omitempty"`
	Price       float64 `json:"price,omitempty" xml:"Price,omitempty"`
}

type MenuItem struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	MajorCategory int        `json:"major_category"`
	MinorCategory int        `json:"minor_category"`
	AltItemNumber int        `json:"alt_item_number,omitempty"`
	Price         float64    `json:"price,omitempty"`
	Prices        []float64  `json:"prices,omitempty"`
	Barcode       string     `json:"barcode,omitempty"`
	Category      *Category  `json:"category,omitempty"`
	Modifiers     []Modifier `json:"modifiers,omitempty"`
}

// --- Order Ticket API Request Models ---
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
	CategoryName string            `json:"category_name,omitempty"`
	Memo         string            `json:"memo,omitempty"`
	Modifiers    []ModifierRequest `json:"modifiers,omitempty"`
}

type ModifierRequest struct {
	ItemNumber string `json:"item_number"`
	ScreenCell string `json:"screen_cell,omitempty"`
	ItemName   string `json:"item_name,omitempty"`
	Quantity   int    `json:"quantity"`
	Memo       string `json:"memo,omitempty"`
}

// --- Order API Response ---
type CreateTicketResponse struct {
	Status          string             `json:"status"`
	ReferenceNumber string             `json:"reference_number,omitempty"`
	Message         string             `json:"message,omitempty"`
	Ticket          *positouch.Ticket  `json:"ticket"`
}

type CreateTicketErrorResponse struct {
	Error           string `json:"error"`
	ReferenceNumber string `json:"reference_number,omitempty"`
}

// --- POSitouch OUT*.XML Confirmation XML models ---
type OrderConfirmation struct {
	XMLName     xml.Name             `xml:"OrderConfirmation"`
	Transaction ConfirmTransaction   `xml:"Transaction"`
}

type ConfirmTransaction struct {
	ReferenceNumber string         `xml:"ReferenceNumber"`
	ResponseCode    int            `xml:"ResponseCode"`
	Error           *ConfirmError  `xml:"Error"`
}

type ConfirmError struct {
	ErrorCode int    `xml:"ErrorCode"`
	Reference int    `xml:"Reference"`
	Text      string `xml:"Text"`
}

// --- POSitouch Ordering XML models ---
type Orders struct {
	XMLName  xml.Name  `xml:"Orders"`
	NewOrder *NewOrder `xml:"NewOrder,omitempty"`
}

type NewOrder struct {
	Function        int    `xml:"Function"`
	ErrorLevel      int    `xml:"ErrorLevel"`
	ReferenceNumber string `xml:"ReferenceNumber,omitempty"`
	Check           *Check `xml:"Check,omitempty"`
}

type Check struct {
	CheckHeader CheckHeader  `xml:"CheckHeader"`
	ItemDetails []ItemDetail `xml:"ItemDetail,omitempty"`
}

type CheckHeader struct {
	TableNumber    string `xml:"TableNumber"`
	CostCenter     string `xml:"CostCenter,omitempty"`
	ServerNumber   string `xml:"ServerNumber"`
	NumberInParty  int    `xml:"NumberInParty,omitempty"`
	TerminalNumber string `xml:"TerminalNumber,omitempty"`
}

type ItemDetail struct {
	ItemNumber   string   `xml:"ItemNumber,omitempty"`
	ScreenCell   string   `xml:"ScreenCell,omitempty"`
	ItemName     string   `xml:"ItemName,omitempty"`
	Quantity     int      `xml:"Quantity"`
	Memo         string   `xml:"Memo,omitempty"`
	CategoryID   string   `xml:"CategoryID,omitempty"`
	CategoryName string   `xml:"CategoryName,omitempty"`
	Options      []Option `xml:"Option,omitempty"`
}

type Option struct {
	ItemNumber string `xml:"ItemNumber,omitempty"`
	ScreenCell string `xml:"ScreenCell,omitempty"`
	ItemName   string `xml:"ItemName,omitempty"`
	Quantity   int    `xml:"Quantity"`
	Memo       string `xml:"Memo,omitempty"`
}

// randomSuffix returns a 6-character lowercase alphanumeric string.
func randomSuffix() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// --- Main API handler for Ticket Creation (local HTTP endpoint) ---

func CreateTicket(w http.ResponseWriter, r *http.Request, inorderDir string, xmlDir string, xmlCloseDir string) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Change 1: Upfront validation — every item must have item_number or screen_cell.
	for i, it := range req.Items {
		if it.ItemNumber == "" && it.ScreenCell == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateTicketErrorResponse{
				Error:           fmt.Sprintf("item at index %d ('%s') has no item_number or screen_cell — POSitouch requires one of these to identify the item", i, it.ItemName),
				ReferenceNumber: req.ReferenceNumber,
			})
			return
		}
		for j, mod := range it.Modifiers {
			if mod.ItemNumber == "" && mod.ScreenCell == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(CreateTicketErrorResponse{
					Error:           fmt.Sprintf("modifier at index %d ('%s') on item %d has no item_number or screen_cell", j, mod.ItemName, i),
					ReferenceNumber: req.ReferenceNumber,
				})
				return
			}
		}
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
			ItemNumber:   it.ItemNumber,
			ScreenCell:   it.ScreenCell,
			ItemName:     it.ItemName,
			Quantity:     it.Quantity,
			Memo:         it.Memo,
			CategoryID:   it.CategoryID,
			CategoryName: it.CategoryName,
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

	if err := writeOrderXMLAtomically(xmlData, inorderDir); err != nil {
		http.Error(w, "failed to write order file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Change 2: Poll xmlDir for OUT*.XML confirmation file matching our ReferenceNumber.
	deadline := time.Now().Add(confirmPollTimeout)

	for time.Now().Before(deadline) {
		conf, confFile, err := findConfirmation(xmlDir, req.ReferenceNumber)
		if err == nil && conf != nil {
			// Clean up the confirmation file.
			if removeErr := os.Remove(confFile); removeErr != nil {
				log.Printf("[orders] warning: failed to remove confirmation file %s: %v", confFile, removeErr)
			}

			if conf.Transaction.ResponseCode != 0 {
				// Error from POSitouch.
				errText := ""
				if conf.Transaction.Error != nil {
					errText = conf.Transaction.Error.Text
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(CreateTicketErrorResponse{
					Error:           errText,
					ReferenceNumber: req.ReferenceNumber,
				})
				return
			}

			// Change 3: Success — find the newly created ticket.
			var matchedTicket *positouch.Ticket
			tableNum, convErr := strconv.Atoi(req.TableNumber)
			if convErr != nil {
				log.Printf("[orders] warning: table_number '%s' is not a valid integer: %v", req.TableNumber, convErr)
			} else {
				tickets, tickErr := positouch.ReadAllTickets(xmlDir, xmlCloseDir)
				if tickErr != nil {
					log.Printf("[orders] warning: failed to read tickets for confirmation: %v", tickErr)
				} else {
					for i := range tickets {
						t := &tickets[i]
						if t.Table == tableNum && time.Since(t.OpenedAt) <= ticketMatchWindow {
							matchedTicket = t
							break
						}
					}
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(CreateTicketResponse{
				Status:          "created",
				ReferenceNumber: req.ReferenceNumber,
				Ticket:          matchedTicket,
			})
			return
		}
		time.Sleep(confirmPollInterval)
	}

	// Change 2: Timeout — no confirmation received within 30 seconds.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGatewayTimeout)
	json.NewEncoder(w).Encode(CreateTicketErrorResponse{
		Error:           "timeout: no confirmation from POSitouch after 30s",
		ReferenceNumber: req.ReferenceNumber,
	})
}

func AddTicketItems(w http.ResponseWriter, r *http.Request, locationID, ticketID string) {
	http.Error(w, "AddTicketItems not yet implemented", http.StatusNotImplemented)
}

func AddTicketPayments(w http.ResponseWriter, r *http.Request, locationID, ticketID string) {
	http.Error(w, "AddTicketPayments not yet implemented", http.StatusNotImplemented)
}

// writeOrderXMLAtomically writes xmlData to a temp file then atomically renames it to
// ORDER<suffix>.XML — the naming convention required by XMLInOrderPath in spcwin.ini.
func writeOrderXMLAtomically(xmlData []byte, dir string) error {
	suffix := randomSuffix()
	tmp := filepath.Join(dir, fmt.Sprintf("ORDER%s.tmp", suffix))
	final := filepath.Join(dir, fmt.Sprintf("ORDER%s.XML", suffix))
	if err := os.WriteFile(tmp, xmlData, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}

// findConfirmation scans xmlDir for OUT*.XML (and OUT*.xml) files and returns the first
// OrderConfirmation whose ReferenceNumber matches refNum, along with the file path.
func findConfirmation(xmlDir, refNum string) (*OrderConfirmation, string, error) {
	patternsUpper, _ := filepath.Glob(filepath.Join(xmlDir, "OUT*.XML"))
	patternsLower, _ := filepath.Glob(filepath.Join(xmlDir, "OUT*.xml"))
	files := append(patternsUpper, patternsLower...)

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var conf OrderConfirmation
		if err := xml.Unmarshal(data, &conf); err != nil {
			continue
		}
		if conf.Transaction.ReferenceNumber == refNum {
			return &conf, f, nil
		}
	}
	return nil, "", fmt.Errorf("not found")
}

// WriteOrderXML builds and atomically writes a POSitouch XML order file to dir.
// Function=1: place order, items not sent to prep yet.
// Function=2: place and send to prep.
// Function=3: place, send and print check.
func WriteOrderXML(req CreateTicketRequest, dir string) error {
	checkHeader := CheckHeader{
		TableNumber:    req.TableNumber,
		CostCenter:     req.CostCenter,
		ServerNumber:   req.ServerNumber,
		NumberInParty:  req.NumberInParty,
		TerminalNumber: req.TerminalNumber,
	}
	order := Orders{
		NewOrder: &NewOrder{
			Function:        2, // place and send to prep
			ErrorLevel:      2,
			ReferenceNumber: req.ReferenceNumber,
			Check: &Check{
				CheckHeader: checkHeader,
			},
		},
	}

	for _, it := range req.Items {
		item := ItemDetail{
			ItemNumber:   it.ItemNumber,
			ScreenCell:   it.ScreenCell,
			ItemName:     it.ItemName,
			Quantity:     it.Quantity,
			Memo:         it.Memo,
			CategoryID:   it.CategoryID,
			CategoryName: it.CategoryName,
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
		return err
	}
	return writeOrderXMLAtomically(xmlData, dir)
}