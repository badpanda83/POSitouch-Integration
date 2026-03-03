package positouch

import "encoding/xml"

// These structs match the XML ordering requirements (outbound order creation)
type Orders struct {
XMLName  xml.Name  `xml:"Orders"`
NewOrder *NewOrder `xml:"NewOrder,omitempty"`
}

type NewOrder struct {
Function        int          `xml:"Function"`
ErrorLevel      int          `xml:"ErrorLevel"`
ReferenceNumber string       `xml:"ReferenceNumber,omitempty"`
Check           *OrderCheck  `xml:"Check"`
}

type OrderCheck struct {
CheckHeader  OrderCheckHeader `xml:"CheckHeader"`
ItemDetails  []OrderItem      `xml:"ItemDetail"`
}

type OrderCheckHeader struct {
TableNumber    string `xml:"TableNumber"`
ServerNumber   string `xml:"ServerNumber"`
CostCenter     string `xml:"CostCenter,omitempty"`
TerminalNumber string `xml:"TerminalNumber,omitempty"`
NumberInParty  int    `xml:"NumberInParty,omitempty"`
}

type OrderItem struct {
	ItemNumber string        `xml:"ItemNumber,omitempty"`
	ScreenCell string        `xml:"ScreenCell,omitempty"`
	ItemName   string        `xml:"ItemName,omitempty"`
	Quantity   int           `xml:"Quantity,omitempty"`
	Memo       string        `xml:"Memo,omitempty"`
	Options    []OrderOption `xml:"Option,omitempty"`
}

type OrderOption struct {
	ItemNumber string `xml:"ItemNumber,omitempty"`
	ScreenCell string `xml:"ScreenCell,omitempty"`
	ItemName   string `xml:"ItemName,omitempty"`
	Quantity   int    `xml:"Quantity,omitempty"`
	Memo       string `xml:"Memo,omitempty"`
}
