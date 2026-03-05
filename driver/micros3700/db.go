package micros3700driver

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	// MySQL driver — imported for its side-effect of registering "mysql".
	_ "github.com/go-sql-driver/mysql"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// openDB opens a connection to the MICROS myq MySQL database.
func openDB(cfg *config.MICROS3700Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseHost, cfg.DatabaseName)
	return sql.Open("mysql", dsn)
}

// SyncEntities reads all MICROS 3700 master data from the myq database and
// returns it as a canonical Snapshot. If the database is unavailable, it logs
// a warning and returns an empty snapshot rather than an error.
func (d *Driver) SyncEntities() (*entities.Snapshot, error) {
	log.Printf("[micros3700] SyncEntities: refreshing master data")

	mcfg := d.cfg.MICROS3700
	if mcfg == nil {
		log.Printf("[micros3700][WARN] micros3700 config is nil; returning empty snapshot")
		return &entities.Snapshot{}, nil
	}

	db, err := openDB(mcfg)
	if err != nil {
		log.Printf("[micros3700][WARN] openDB: %v", err)
		return &entities.Snapshot{}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[micros3700][WARN] database ping failed: %v; returning empty snapshot", err)
		return &entities.Snapshot{}, nil
	}

	employees, err := syncEmployees(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncEmployees: %v", err)
	}

	tables, err := syncTables(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncTables: %v", err)
	}

	tenders, err := syncTenders(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncTenders: %v", err)
	}

	costCenters, err := syncCostCenters(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncCostCenters: %v", err)
	}

	menuItems, err := syncMenuItems(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncMenuItems: %v", err)
	}

	orderTypes, err := syncOrderTypes(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncOrderTypes: %v", err)
	}

	return &entities.Snapshot{
		Employees:   employees,
		Tables:      tables,
		Tenders:     tenders,
		CostCenters: costCenters,
		MenuItems:   menuItems,
		OrderTypes:  orderTypes,
	}, nil
}

func syncEmployees(db *sql.DB) ([]entities.Employee, error) {
	rows, err := db.Query(`SELECT obj_num, first_name, last_name, emp_type, mag_card_id FROM emp`)
	if err != nil {
		return nil, fmt.Errorf("query emp: %w", err)
	}
	defer rows.Close()

	var out []entities.Employee
	for rows.Next() {
		var objNum, empType, magCardID int
		var firstName, lastName string
		if err := rows.Scan(&objNum, &firstName, &lastName, &empType, &magCardID); err != nil {
			return nil, fmt.Errorf("scan emp row: %w", err)
		}
		out = append(out, entities.Employee{
			ID:        strconv.Itoa(objNum),
			FirstName: firstName,
			LastName:  lastName,
			Type:      empType,
			Store:     "micros",
			MagCardID: magCardID,
		})
	}
	return out, rows.Err()
}

func syncTables(db *sql.DB) ([]entities.Table, error) {
	rows, err := db.Query(`SELECT obj_num, tbl_name, rvc_obj_num FROM dtbl`)
	if err != nil {
		return nil, fmt.Errorf("query dtbl: %w", err)
	}
	defer rows.Close()

	var out []entities.Table
	for rows.Next() {
		var objNum, rvcObjNum int
		var tblName string
		if err := rows.Scan(&objNum, &tblName, &rvcObjNum); err != nil {
			return nil, fmt.Errorf("scan dtbl row: %w", err)
		}
		out = append(out, entities.Table{
			ID:         strconv.Itoa(objNum),
			Name:       tblName,
			CostCenter: rvcObjNum,
		})
	}
	return out, rows.Err()
}

func syncTenders(db *sql.DB) ([]entities.Tender, error) {
	rows, err := db.Query(`SELECT obj_num, tndr_name, tndr_num FROM tndr`)
	if err != nil {
		return nil, fmt.Errorf("query tndr: %w", err)
	}
	defer rows.Close()

	var out []entities.Tender
	for rows.Next() {
		var objNum, tndrNum int
		var tndrName string
		if err := rows.Scan(&objNum, &tndrName, &tndrNum); err != nil {
			return nil, fmt.Errorf("scan tndr row: %w", err)
		}
		out = append(out, entities.Tender{
			ID:   strconv.Itoa(objNum),
			Name: tndrName,
			Code: strconv.Itoa(tndrNum),
		})
	}
	return out, rows.Err()
}

func syncCostCenters(db *sql.DB) ([]entities.CostCenter, error) {
	rows, err := db.Query(`SELECT obj_num, rvc_name FROM rvc`)
	if err != nil {
		return nil, fmt.Errorf("query rvc: %w", err)
	}
	defer rows.Close()

	var out []entities.CostCenter
	for rows.Next() {
		var objNum int
		var rvcName string
		if err := rows.Scan(&objNum, &rvcName); err != nil {
			return nil, fmt.Errorf("scan rvc row: %w", err)
		}
		out = append(out, entities.CostCenter{
			ID:   strconv.Itoa(objNum),
			Name: rvcName,
		})
	}
	return out, rows.Err()
}

func syncMenuItems(db *sql.DB) ([]entities.MenuItem, error) {
	rows, err := db.Query(`SELECT obj_num, mi_name, mi_num, major_grp_obj_num, family_grp_obj_num,
		price1, price2, price3, price4, price5, price6, price7, price8, price9 FROM mi`)
	if err != nil {
		return nil, fmt.Errorf("query mi: %w", err)
	}
	defer rows.Close()

	var out []entities.MenuItem
	for rows.Next() {
		var objNum, miNum, majorGrp, familyGrp int
		var miName string
		var prices [9]float64
		if err := rows.Scan(&objNum, &miName, &miNum, &majorGrp, &familyGrp,
			&prices[0], &prices[1], &prices[2], &prices[3], &prices[4],
			&prices[5], &prices[6], &prices[7], &prices[8]); err != nil {
			return nil, fmt.Errorf("scan mi row: %w", err)
		}
		priceSlice := prices[:]
		out = append(out, entities.MenuItem{
			ID:            objNum,
			Name:          miName,
			AltItemNumber: miNum,
			Price:         prices[0],
			Prices:        priceSlice,
			MajorCategory: majorGrp,
			MinorCategory: familyGrp,
		})
	}
	return out, rows.Err()
}

func syncOrderTypes(db *sql.DB) ([]entities.OrderType, error) {
	rows, err := db.Query(`SELECT obj_num, ordtype_name FROM ordtype`)
	if err != nil {
		return nil, fmt.Errorf("query ordtype: %w", err)
	}
	defer rows.Close()

	var out []entities.OrderType
	for rows.Next() {
		var objNum int
		var ordtypeName string
		if err := rows.Scan(&objNum, &ordtypeName); err != nil {
			return nil, fmt.Errorf("scan ordtype row: %w", err)
		}
		out = append(out, entities.OrderType{
			ID:   strconv.Itoa(objNum),
			Name: ordtypeName,
		})
	}
	return out, rows.Err()
}
