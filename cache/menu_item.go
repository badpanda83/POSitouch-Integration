package cache

// Category represents a menu category (e.g., appetizers, drinks, desserts)
type Category struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    Major       int    `json:"major,omitempty"` // e.g., Major category (from POSitouch)
    Minor       int    `json:"minor,omitempty"` // e.g., Minor category (from POSitouch)
}

// Modifier represents a menu modifier (e.g., extra cheese, no onions)
type Modifier struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    PriceChange float64 `json:"price_change,omitempty"`
}

// MenuItem represents a POS menu item, e.g., food or drink
type MenuItem struct {
    ID             int        `json:"id"`                // Item Number from POSitouch
    Name           string     `json:"name"`              // Description
    MajorCategory  int        `json:"major_category"`    // Major
    MinorCategory  int        `json:"minor_category"`    // Minor
    AltItemNumber  int        `json:"alt_item_number,omitempty"`
    Price          float64    `json:"price"`
    Prices         []float64  `json:"prices,omitempty"`  // Support for multiple price levels
    CategoryIDs    []int      `json:"category_ids"`      // IDs this item is part of (usually 1 cat)
    ModifierIDs    []int      `json:"modifier_ids"`      // IDs of allowed modifiers
    Description    string     `json:"description,omitempty"`
    Barcode        string     `json:"barcode,omitempty"`
}
// Dummy/placeholder methods; these let the integration compile and
// return empty lists for now. You can improve them later.

func ReadMenuItems(dbfDir string) ([]MenuItem, error) {
    // TODO: Replace with real DBF parsing!
    return []MenuItem{}, nil
}

func ReadModifiers(dbfDir string) ([]Modifier, error) {
    // TODO: Replace with real DBF parsing!
    return []Modifier{}, nil
}

func ReadCategories(dbfDir string) ([]Category, error) {
    // TODO: Replace with real DBF parsing!
    return []Category{}, nil
}