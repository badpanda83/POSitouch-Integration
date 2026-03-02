package positouch

// Category represents a menu category (e.g., appetizers, drinks, desserts)
type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Major       int    `json:"major"`
	Minor       int    `json:"minor"`
}

// Modifier represents a menu modifier (e.g., extra cheese, no onions)
type Modifier struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	PriceChange float64 `json:"price_change,omitempty"`
}

// MenuItem represents a POS menu item, e.g., food or drink
type MenuItem struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	MajorCategory int       `json:"major_category"`
	MinorCategory int       `json:"minor_category"`
	AltItemNumber int       `json:"alt_item_number,omitempty"`
	Price         float64   `json:"price"`
	Prices        []float64 `json:"prices,omitempty"`
	Barcode       string    `json:"barcode,omitempty"`
	CategoryIDs   []int     `json:"category_ids,omitempty"`
	ModifierIDs   []int     `json:"modifier_ids,omitempty"`
}