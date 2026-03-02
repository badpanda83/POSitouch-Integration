package main

import (
	"fmt"
	"log"

	"github.com/badpanda83/POSitouch-Integration/positouch"
)

func main() {
	menuItems, err := positouch.ParseMenuXML("C:/Users/Omnivore/Documents/POSitouch-Integration/utils/Export/menu_items.xml")
	if err != nil {
		log.Fatalf("ParseMenuXML error: %v", err)
	}
	fmt.Printf("Parsed %d menu items\n", len(menuItems))
	for i, item := range menuItems {
		if i >= 10 { break } // Only print first 10
		fmt.Printf("Item: %+v\n", item)
	}
}