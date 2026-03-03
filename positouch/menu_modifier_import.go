package positouch

import (
"encoding/xml"
"log"
"os"
)

// ParseMenuModifiers extracts modifier items (MajorCategory 11 = FOOD OPTIONS)
// from the same menu_items.xml produced by WExport.
func ParseMenuModifiers(filename string) ([]Modifier, error) {
f, err := os.Open(filename)
if err != nil {
return nil, err
}
defer f.Close()

var doc indataDbfXML
if err := xml.NewDecoder(f).Decode(&doc); err != nil {
return nil, err
}

var modifiers []Modifier
for _, x := range doc.MenuItems {
if x.MajorCategory != 11 {
continue
}
if x.Description == "" {
continue
}
modifiers = append(modifiers, Modifier{
ID:          x.ItemNumber,
Name:        x.Description,
Description: x.ExtendedDescription,
PriceChange: x.Price1,
})
}
log.Printf("[positouch] parsed %d modifiers (MajorCategory=11) from %s", len(modifiers), filename)
return modifiers, nil
}
