package positouch

import (
    "encoding/xml"
    "os"
    "strconv"
    "strings"
)

type fldflagsXML struct {
    XMLName       xml.Name       `xml:"ItemMaintenance"`
    StoreNumber   int            `xml:"StoreNumber"`
    FieldsAndFlags fieldsAndFlags `xml:"FieldsAndFlags"`
}

type fieldsAndFlags struct {
    MajorMinorCategoryTable majorMinorCategoryTable `xml:"MajorMinorCategoryTable"`
}

type majorMinorCategoryTable struct {
    // Catch-all for dynamically named elements
    Fields map[string]string `xml:",any"`
}

// ParseFLDFLAGS parses FLDFLAGS.xml and returns categories as a slice.
func ParseFLDFLAGS(path string) ([]MenuCategory, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    dec := xml.NewDecoder(f)
    var doc fldflagsXML
    if err := dec.Decode(&doc); err != nil {
        return nil, err
    }

    cats := []MenuCategory{}
    mmct := doc.FieldsAndFlags.MajorMinorCategoryTable

    // Parse major/minor categories from keys like Major1, Major1Minor1 ...
    for i := 1; i <= 20; i++ {
        majorKey := "Major" + strconv.Itoa(i)
        majorName := strings.TrimSpace(mmct.Fields[majorKey])
        if majorName == "" {
            continue // skip unused majors
        }
        // Always include the major as a "category" by itself (for main listing)
        cats = append(cats, MenuCategory{
            MajorNum:  i,
            MajorName: majorName,
            MinorNum:  0,
            MinorName: "",
        })

        for j := 1; j <= 10; j++ {
            minorKey := majorKey + "Minor" + strconv.Itoa(j)
            minorName := strings.TrimSpace(mmct.Fields[minorKey])
            if minorName != "" {
                cats = append(cats, MenuCategory{
                    MajorNum:  i,
                    MajorName: majorName,
                    MinorNum:  j,
                    MinorName: minorName,
                })
            }
        }
    }
    return cats, nil
}