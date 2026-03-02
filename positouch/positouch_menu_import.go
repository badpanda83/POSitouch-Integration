package positouch

import (
	"encoding/xml"
	"os"
)

// For categories
type categoryXML struct {
	ID          int    `xml:"ID"`
	Name        string `xml:"Name"`
	Description string `xml:"Description"`
	Major       int    `xml:"Major"`
	Minor       int    `xml:"Minor"`
}

type categoryDbfXML struct {
	Categories []categoryXML `xml:"Category"`
}

// ParseMenuCategories parses your menu_categories.xml into []Category.
func ParseMenuCategories(filename string) ([]Category, error) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	var dbf categoryDbfXML
	dec := xml.NewDecoder(xmlFile)
	if err := dec.Decode(&dbf); err != nil {
		return nil, err
	}

	var cats []Category
	for _, x := range dbf.Categories {
		cats = append(cats, Category{
			ID:          x.ID,
			Name:        x.Name,
			Description: x.Description,
			Major:       x.Major,
			Minor:       x.Minor,
		})
	}
	return cats, nil
}

// Your MenuItem importer (unchanged):
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

// ParseMenuXML parses the menu XML and returns a slice of canonical MenuItems
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