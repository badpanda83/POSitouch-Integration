// Package entities defines canonical, POS-agnostic data types shared across all
// POS driver implementations.
package entities

// Snapshot is the full set of POS master data returned by SyncEntities.
type Snapshot struct {
	Tables      []Table
	Employees   []Employee
	Tenders     []Tender
	CostCenters []CostCenter
	OrderTypes  []OrderType
	MenuItems   []MenuItem
}

// Table represents a restaurant table.
type Table struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Section    string `json:"section,omitempty"`
	CostCenter int    `json:"cost_center,omitempty"`
}

// Employee represents a POS employee/server.
type Employee struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Type      int    `json:"type"`
	Store     string `json:"store"`
	MagCardID int    `json:"mag_card_id,omitempty"`
}

// Tender represents a payment type.
type Tender struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// CostCenter represents a revenue center or cost center.
type CostCenter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OrderType represents an order/revenue type.
type OrderType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MenuItem represents a POS menu item.
type MenuItem struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	MajorCategory int       `json:"major_category"`
	MinorCategory int       `json:"minor_category"`
	AltItemNumber int       `json:"alt_item_number,omitempty"`
	Price         float64   `json:"price,omitempty"`
	Prices        []float64 `json:"prices,omitempty"`
	Barcode       string    `json:"barcode,omitempty"`
}

// TicketItemOption represents a modifier/option applied to a ticket item.
type TicketItemOption struct {
	ItemNumber   int    `json:"item_number"`
	ItemName     string `json:"item_name"`
	CellName     string `json:"cell_name"`
	Memo         string `json:"memo"`
	MajorName    string `json:"major_name"`
	MajorNumber  int    `json:"major_number"`
	MinorName    string `json:"minor_name"`
	MinorNumber  int    `json:"minor_number"`
	ScreenNumber int    `json:"screen_number"`
	CellNumber   int    `json:"cell_number"`
}

// TicketItem represents a single item on a ticket.
type TicketItem struct {
	ItemNumber       int               `json:"item_number"`
	ItemName         string            `json:"item_name"`
	CellName         string            `json:"cell_name"`
	MajorName        string            `json:"major_name"`
	MajorNumber      int               `json:"major_number"`
	MinorName        string            `json:"minor_name"`
	MinorNumber      int               `json:"minor_number"`
	ScreenNumber     int               `json:"screen_number"`
	CellNumber       int               `json:"cell_number"`
	FullPrice        float64           `json:"full_price"`
	NetPrice         float64           `json:"net_price"`
	PriceLevel       int               `json:"price_level"`
	SendTime         string            `json:"send_time"`
	SerialNumber     int               `json:"serial_number"`
	PrepSequence     int               `json:"prep_sequence"`
	PrepSequenceName string            `json:"prep_sequence_name"`
	SplitItem        string            `json:"split_item"`
	SentToPrep       string            `json:"sent_to_prep"`
	NewServer        int               `json:"new_server"`
	NewServerName    string            `json:"new_server_name"`
	Option           *TicketItemOption `json:"option,omitempty"`
}

// Ticket represents a POS check/ticket.
type Ticket struct {
	Number         int          `json:"number"`
	OpenedAt       string       `json:"opened_at"`
	Table          int          `json:"table"`
	Total          float64      `json:"total"`
	ServerName     string       `json:"server_name"`
	PartySize      int          `json:"party_size"`
	CostCenter     int          `json:"cost_center"`
	CostCenterName string       `json:"cost_center_name"`
	ServiceCharge  float64      `json:"service_charge,omitempty"`
	Discount       float64      `json:"discount,omitempty"`
	Items          []TicketItem `json:"items"`
	Open           bool         `json:"open"`
	POSType        string       `json:"pos_type,omitempty"`
}

// ModifierRequest is an order modifier (option) within a CreateOrderRequest.
type ModifierRequest struct {
	ItemNumber string `json:"item_number"`
	ScreenCell string `json:"screen_cell,omitempty"`
	ItemName   string `json:"item_name,omitempty"`
	Quantity   int    `json:"quantity"`
	Memo       string `json:"memo,omitempty"`
}

// OrderItemRequest is a single item within a CreateOrderRequest.
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

// CreateOrderRequest is the payload used to place a new order through a POSDriver.
type CreateOrderRequest struct {
	ReferenceNumber string             `json:"reference_number"`
	TableNumber     string             `json:"table_number"`
	CostCenter      string             `json:"cost_center,omitempty"`
	ServerNumber    string             `json:"server_number"`
	TerminalNumber  string             `json:"terminal_number,omitempty"`
	NumberInParty   int                `json:"number_in_party,omitempty"`
	Items           []OrderItemRequest `json:"items"`
}

// CreateOrderResponse is the successful response from CreateOrder.
type CreateOrderResponse struct {
	Status          string  `json:"status"`
	ReferenceNumber string  `json:"reference_number,omitempty"`
	Message         string  `json:"message,omitempty"`
	Ticket          *Ticket `json:"ticket"`
}

// CreateOrderErrorResponse is the error response from CreateOrder.
type CreateOrderErrorResponse struct {
	Error           string `json:"error"`
	ReferenceNumber string `json:"reference_number,omitempty"`
}
