package positouch

import "encoding/xml"

// These structs match the XML ordering requirements
type Orders struct {
    XMLName    xml.Name    `xml:"Orders"`
    NewOrder   *NewOrder   `xml:"NewOrder,omitempty"`
}

type NewOrder struct {
    Function        int        `xml:"Function"`
    ErrorLevel      int        `xml:"ErrorLevel"`
    ReferenceNumber string     `xml:"ReferenceNumber,omitempty"`
    Check           Check      `xml:"Check"`
}

type Check struct {
    CheckHeader  CheckHeader    `xml:"CheckHeader"`
    ItemDetails  []ItemDetail   `xml:"ItemDetail"`
}

type CheckHeader struct {
    TableNumber   string   `xml:"TableNumber"`
    ServerNumber  string   `xml:"ServerNumber"`
    // Add more tags as needed (CostCenter, TerminalNumber, etc)
}

type ItemDetail struct {
    ItemNumber string      `xml:"ItemNumber"`
    Quantity   int         `xml:"Quantity,omitempty"`
    Options    []Option    `xml:"Option,omitempty"`             // For modifiers (set modifiers etc.)
}

type Option struct {
    ItemNumber string `xml:"ItemNumber"`
}