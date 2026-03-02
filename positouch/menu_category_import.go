package positouch

import (
	"encoding/xml"
	"log"
	"os"
	"strconv"
	"strings"
)

type itemMaintenanceXML struct {
	XMLName        xml.Name       `xml:"ItemMaintenance"`
	FieldsAndFlags fieldsAndFlags `xml:"FieldsAndFlags"`
}

type fieldsAndFlags struct {
	MajorMinorCategoryTable majorMinorCategoryTable `xml:"MajorMinorCategoryTable"`
}

type xmlKV struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type majorMinorCategoryTable struct {
	Fields []xmlKV `xml:",any"`
}

func ParseMenuCategories(path string) ([]Category, error) {
	log.Printf("[menu_category_import] Opening category XML: %s", path)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[menu_category_import] ERROR opening file: %v", err)
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var doc itemMaintenanceXML
	if err := dec.Decode(&doc); err != nil {
		log.Printf("[menu_category_import] ERROR decoding XML: %v", err)
		return nil, err
	}

	fieldMap := make(map[string]string)
	for _, kv := range doc.FieldsAndFlags.MajorMinorCategoryTable.Fields {
		fieldMap[kv.XMLName.Local] = strings.TrimSpace(kv.Value)
	}

	var categories []Category

	for major := 1; major <= 20; major++ {
		majorKey := "Major" + strconv.Itoa(major)
		majorName := fieldMap[majorKey]
		if majorName == "" {
			continue
		}
		categories = append(categories, Category{
			ID:          major * 1000,
			Name:        majorName,
			Description: "",
			Major:       major,
			Minor:       0,
		})
		for minor := 1; minor <= 10; minor++ {
			minorKey := majorKey + "Minor" + strconv.Itoa(minor)
			minorName := fieldMap[minorKey]
			if minorName != "" {
				categories = append(categories, Category{
					ID:          major*1000 + minor,
					Name:        minorName,
					Description: "",
					Major:       major,
					Minor:       minor,
				})
			}
		}
	}
	log.Printf("[menu_category_import] Parsed %d categories from %s", len(categories), path)
	return categories, nil
}