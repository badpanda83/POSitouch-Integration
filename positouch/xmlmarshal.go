package positouch

import (
    "encoding/xml"
)

func MarshalOrderToXML(order *Orders) ([]byte, error) {
    return xml.MarshalIndent(order, "", "  ")
}