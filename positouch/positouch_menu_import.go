package positouch

import (
	"encoding/xml"
	"os"
)

type menuItemXML struct {
	ItemNumber          int     `xml:"ItemNumber"`
	Description         string  `xml:"Description"`
	ExtendedDescription string  `xml:"ExtendedDescription"`
	Price1              float64 `xml:"Price1"`
	Price2              float64 `xml:"Price2"`
	Price3              float64 `xml:"Price3"`
	Price4              float64 `xml:"Price4"`
	Price5              float64 `xml:"Price5"`
	Price6              float64 `xml:"Price6"`
	Price7              float64 `xml:"Price7"`
	Price8              float64 `xml:"Price8"`
	Price9              float64 `xml:"Price9"`
	Price10             float64 `xml:"Price10"`
	MajorCategory       int     `xml:"MajorCategory"`
	MinorCategory       int     `xml:"MinorCategory"`
	AltItemNumber       int     `xml:"AlternateItemNumber"`
	Barcode             string  `xml:"Barcode"`
}

type indataDbfXML struct {
	XMLName     xml.Name      `xml:"IndataDbf"`
	StoreNumber int           `xml:"StoreNumber"`
	MenuItems   []menuItemXML `xml:"MenuItem"`
}

// ParseMenuXML parses the menu XML and returns a slice of canonical MenuItems.
func ParseMenuXML(filename string) ([]MenuItem, error) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	var dbf indataDbfXML
	dec := xml.NewDecoder(xmlFile)
	if err := dec.Decode(&dbf); err != nil {
		return nil, err
	}

	var items []MenuItem
	for _, x := range dbf.MenuItems {
		it := MenuItem{
			ID:            x.ItemNumber,
			Name:          x.Description,
			Description:   x.ExtendedDescription,
			MajorCategory: x.MajorCategory,
			MinorCategory: x.MinorCategory,
			AltItemNumber: x.AltItemNumber,
			Price:         x.Price1,
			Prices: []float64{
				x.Price1, x.Price2, x.Price3, x.Price4, x.Price5,
				x.Price6, x.Price7, x.Price8, x.Price9, x.Price10,
			},
			Barcode: x.Barcode,
		}
		items = append(items, it)
	}
	return items, nil
}