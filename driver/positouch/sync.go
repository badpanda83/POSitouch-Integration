package positouchdriver

import (
	"fmt"
	"log"
	"strconv"

	"github.com/badpanda83/POSitouch-Integration/entities"
	"github.com/badpanda83/POSitouch-Integration/positouch"
)

const (
	exportDir = `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export`
	tablesXML = exportDir + `\set1.xml`
)

// SyncEntities reads all POSitouch master data and returns it as a canonical Snapshot.
func (d *Driver) SyncEntities() (*entities.Snapshot, error) {
	log.Printf("[positouch] SyncEntities: refreshing master data")

	// Regenerate and copy set1.xml so table data is fresh.
	if err := positouch.RunWExportAndCopySet1(); err != nil {
		log.Printf("[positouch][WARN] WExport failed, tables may be stale: %v", err)
	} else {
		log.Printf("[positouch] WExport completed, set1.xml refreshed")
	}

	rawEmployees, err := positouch.ReadEmployees(d.cfg.DBFDir, d.cfg.SCDir)
	if err != nil {
		log.Printf("[positouch][WARN] ReadEmployees: %v", err)
	}

	rawOrderTypes, err := positouch.ReadOrderTypes(d.cfg.DBFDir)
	if err != nil {
		log.Printf("[positouch][WARN] ReadOrderTypes: %v", err)
	}

	rawTenders, err := positouch.ReadTenders(d.cfg.DBFDir)
	if err != nil {
		log.Printf("[positouch][WARN] ReadTenders: %v", err)
	}

	rawCostCenters, err := positouch.ReadCostCenters(d.cfg.DBFDir)
	if err != nil {
		log.Printf("[positouch][WARN] ReadCostCenters: %v", err)
	}

	rawTables, tableErr := positouch.ParseTablesFromSet1XML(tablesXML)
	if tableErr != nil {
		log.Printf("[positouch][WARN] ParseTablesFromSet1XML: %v", tableErr)
	}

	menuXMLPath := fmt.Sprintf(`%s\menu_items.xml`, exportDir)
	rawMenuItems, err := positouch.ParseMenuXML(menuXMLPath)
	if err != nil {
		log.Printf("[positouch][WARN] ParseMenuXML: %v", err)
	}

	snapshot := &entities.Snapshot{
		Employees:   mapEmployees(rawEmployees),
		OrderTypes:  mapOrderTypes(rawOrderTypes),
		Tenders:     mapTenders(rawTenders),
		CostCenters: mapCostCenters(rawCostCenters),
		Tables:      mapTables(rawTables),
		MenuItems:   mapMenuItems(rawMenuItems),
	}
	return snapshot, nil
}

func mapEmployees(src []positouch.Employee) []entities.Employee {
	out := make([]entities.Employee, 0, len(src))
	for _, e := range src {
		out = append(out, entities.Employee{
			ID:        strconv.Itoa(e.Number),
			FirstName: e.FirstName,
			LastName:  e.LastName,
			Type:      e.Type,
			Store:     e.Store,
			MagCardID: e.MagCardID,
		})
	}
	return out
}

func mapOrderTypes(src []positouch.OrderType) []entities.OrderType {
	out := make([]entities.OrderType, 0, len(src))
	for _, ot := range src {
		out = append(out, entities.OrderType{
			ID:   strconv.Itoa(ot.ID),
			Name: ot.Name,
		})
	}
	return out
}

func mapTenders(src []positouch.Tender) []entities.Tender {
	out := make([]entities.Tender, 0, len(src))
	for _, t := range src {
		code := strconv.Itoa(t.Code)
		out = append(out, entities.Tender{
			ID:   code,
			Name: t.Name,
			Code: code,
		})
	}
	return out
}

func mapCostCenters(src []positouch.CostCenter) []entities.CostCenter {
	out := make([]entities.CostCenter, 0, len(src))
	for _, cc := range src {
		out = append(out, entities.CostCenter{
			ID:   strconv.Itoa(cc.Code),
			Name: cc.Name,
		})
	}
	return out
}

func mapTables(src []positouch.Table) []entities.Table {
	out := make([]entities.Table, 0, len(src))
	for _, t := range src {
		out = append(out, entities.Table{
			ID:         strconv.Itoa(t.ID),
			Name:       strconv.Itoa(t.ID),
			Section:    t.RevenueCenter,
			CostCenter: t.RevenueCenterID,
		})
	}
	return out
}

func mapMenuItems(src []positouch.MenuItem) []entities.MenuItem {
	out := make([]entities.MenuItem, 0, len(src))
	for _, m := range src {
		out = append(out, entities.MenuItem{
			ID:            m.ID,
			Name:          m.Name,
			Description:   m.Description,
			MajorCategory: m.MajorCategory,
			MinorCategory: m.MinorCategory,
			AltItemNumber: m.AltItemNumber,
			Price:         m.Price,
			Prices:        m.Prices,
			Barcode:       m.Barcode,
		})
	}
	return out
}
