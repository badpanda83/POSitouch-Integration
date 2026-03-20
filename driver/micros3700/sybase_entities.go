//go:build windows && cgo

package micros3700driver

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/alexbrainman/odbc"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

// storeID is the store identifier tag applied to all entities from this driver.
const storeID = "micros"

// odbcDSN returns the effective ODBC DSN from the config, defaulting to "Micros".
func odbcDSN(cfg *config.Config) string {
	if cfg.MICROS3700 != nil && cfg.MICROS3700.ODBCDSN != "" {
		return cfg.MICROS3700.ODBCDSN
	}
	return "Micros"
}

// syncEntitiesODBC reads all MICROS 3700 master data from the Sybase SQL
// Anywhere 16 database through the configured ODBC DSN.
func syncEntitiesODBC(cfg *config.Config) (*entities.Snapshot, error) {
	log.Printf("[micros3700] SyncEntities: refreshing master data via ODBC")

	dsn := odbcDSN(cfg)
	db, err := sql.Open("odbc", fmt.Sprintf("DSN=%s", dsn))
	if err != nil {
		log.Printf("[micros3700][WARN] ODBC open (DSN=%s): %v — returning empty snapshot; "+
			"ensure github.com/alexbrainman/odbc is imported in the Windows build", dsn, err)
		return &entities.Snapshot{}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[micros3700][WARN] ODBC ping (DSN=%s): %v — returning empty snapshot", dsn, err)
		return &entities.Snapshot{}, nil
	}

	employees, err := odbcEmployees(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcEmployees: %v", err)
	}

	tables, err := odbcTables(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcTables: %v", err)
	}

	tenders, err := odbcTenders(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcTenders: %v", err)
	}

	costCenters, err := odbcCostCenters(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcCostCenters: %v", err)
	}

	orderTypes, err := odbcOrderTypes(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcOrderTypes: %v", err)
	}

	menuItems, err := odbcMenuItems(db)
	if err != nil {
		log.Printf("[micros3700][WARN] odbcMenuItems: %v", err)
	}

	return &entities.Snapshot{
		Employees:   employees,
		Tables:      tables,
		Tenders:     tenders,
		CostCenters: costCenters,
		OrderTypes:  orderTypes,
		MenuItems:   menuItems,
	}, nil
}

// syncTicketsODBC reads all open checks from the gst_chk_hdr table.
func syncTicketsODBC(cfg *config.Config) ([]entities.Ticket, error) {
	log.Printf("[micros3700] SyncTickets: reading open checks via ODBC")

	dsn := odbcDSN(cfg)
	db, err := sql.Open("odbc", fmt.Sprintf("DSN=%s", dsn))
	if err != nil {
		log.Printf("[micros3700][WARN] ODBC open (DSN=%s): %v — returning empty ticket list", dsn, err)
		return []entities.Ticket{}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[micros3700][WARN] ODBC ping (DSN=%s): %v — returning empty ticket list", dsn, err)
		return []entities.Ticket{}, nil
	}

	rows, err := db.Query(`SELECT h.chk_seq, h.rvc_seq, h.tab_seq, h.chk_ttl, h.chk_opn_tim FROM gst_chk_hdr h WHERE h.chk_clsd_flag = 0 ORDER BY h.chk_seq`)
	if err != nil {
		return nil, fmt.Errorf("query gst_chk_hdr: %w", err)
	}
	defer rows.Close()

	var out []entities.Ticket
	for rows.Next() {
		var chkSeq, rvcSeq, tabSeq, chkTtl int
		var chkOpnTim time.Time
		if err := rows.Scan(&chkSeq, &rvcSeq, &tabSeq, &chkTtl, &chkOpnTim); err != nil {
			return nil, fmt.Errorf("scan gst_chk_hdr row: %w", err)
		}
		out = append(out, entities.Ticket{
			Number:     chkSeq,
			Table:      tabSeq,
			CostCenter: rvcSeq,
			Total:      float64(chkTtl) / 100.0,
			OpenedAt:   chkOpnTim.Format(time.RFC3339),
			Open:       true,
			POSType:    "micros3700",
		})
	}
	return out, rows.Err()
}

func odbcEmployees(db *sql.DB) ([]entities.Employee, error) {
	rows, err := db.Query(`SELECT emp_seq, LTRIM(RTRIM(emp_last_name)) || ', ' || LTRIM(RTRIM(emp_first_name)) as full_name FROM emp WHERE emp_active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query emp: %w", err)
	}
	defer rows.Close()

	var out []entities.Employee
	for rows.Next() {
		var empSeq int
		var fullName string
		if err := rows.Scan(&empSeq, &fullName); err != nil {
			return nil, fmt.Errorf("scan emp row: %w", err)
		}
		// Store the combined "Last, First" name in LastName for display.
		out = append(out, entities.Employee{
			ID:       strconv.Itoa(empSeq),
			LastName: fullName,
			Store:    storeID,
		})
	}
	return out, rows.Err()
}

func odbcTables(db *sql.DB) ([]entities.Table, error) {
	rows, err := db.Query(`SELECT obj_num, LTRIM(RTRIM(name)) as name FROM tab WHERE active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query tab: %w", err)
	}
	defer rows.Close()

	var out []entities.Table
	for rows.Next() {
		var objNum int
		var name string
		if err := rows.Scan(&objNum, &name); err != nil {
			return nil, fmt.Errorf("scan tab row: %w", err)
		}
		out = append(out, entities.Table{
			ID:   strconv.Itoa(objNum),
			Name: name,
		})
	}
	return out, rows.Err()
}

func odbcTenders(db *sql.DB) ([]entities.Tender, error) {
	rows, err := db.Query(`SELECT obj_num, LTRIM(RTRIM(name)) as name FROM tnd WHERE active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query tnd: %w", err)
	}
	defer rows.Close()

	var out []entities.Tender
	for rows.Next() {
		var objNum int
		var name string
		if err := rows.Scan(&objNum, &name); err != nil {
			return nil, fmt.Errorf("scan tnd row: %w", err)
		}
		out = append(out, entities.Tender{
			ID:   strconv.Itoa(objNum),
			Name: name,
		})
	}
	return out, rows.Err()
}

func odbcCostCenters(db *sql.DB) ([]entities.CostCenter, error) {
	rows, err := db.Query(`SELECT obj_num, LTRIM(RTRIM(name)) as name FROM rvc WHERE active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query rvc: %w", err)
	}
	defer rows.Close()

	var out []entities.CostCenter
	for rows.Next() {
		var objNum int
		var name string
		if err := rows.Scan(&objNum, &name); err != nil {
			return nil, fmt.Errorf("scan rvc row: %w", err)
		}
		out = append(out, entities.CostCenter{
			ID:   strconv.Itoa(objNum),
			Name: name,
		})
	}
	return out, rows.Err()
}

func odbcOrderTypes(db *sql.DB) ([]entities.OrderType, error) {
	rows, err := db.Query(`SELECT obj_num, LTRIM(RTRIM(name)) as name FROM ord_type WHERE active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query ord_type: %w", err)
	}
	defer rows.Close()

	var out []entities.OrderType
	for rows.Next() {
		var objNum int
		var name string
		if err := rows.Scan(&objNum, &name); err != nil {
			return nil, fmt.Errorf("scan ord_type row: %w", err)
		}
		out = append(out, entities.OrderType{
			ID:   strconv.Itoa(objNum),
			Name: name,
		})
	}
	return out, rows.Err()
}

func odbcMenuItems(db *sql.DB) ([]entities.MenuItem, error) {
	rows, err := db.Query(`SELECT mi_seq, LTRIM(RTRIM(name1)) as name, def_price FROM mi_def WHERE active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query mi_def: %w", err)
	}
	defer rows.Close()

	var out []entities.MenuItem
	for rows.Next() {
		var miSeq int
		var name string
		var defPrice float64
		if err := rows.Scan(&miSeq, &name, &defPrice); err != nil {
			return nil, fmt.Errorf("scan mi_def row: %w", err)
		}
		out = append(out, entities.MenuItem{
			ID:    miSeq,
			Name:  name,
			Price: defPrice,
		})
	}
	return out, rows.Err()
}