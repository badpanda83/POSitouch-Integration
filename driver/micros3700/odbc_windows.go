//go:build windows && cgo

package micros3700driver

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	// ODBC driver — imported for its side-effect of registering "odbc".
	_ "github.com/alexbrainman/odbc"

	"github.com/badpanda83/POSitouch-Integration/config"
	"github.com/badpanda83/POSitouch-Integration/entities"
)

func openODBC(cfg *config.Config) (*sql.DB, error) {
	dsn := "DSN=Micros"
	if cfg.MICROS3700 != nil && cfg.MICROS3700.ODBCDSN != "" {
		dsn = "DSN=" + cfg.MICROS3700.ODBCDSN
	}
	return sql.Open("odbc", dsn)
}

func syncEntitiesODBC(cfg *config.Config) (*entities.Snapshot, error) {
	log.Printf("[micros3700] SyncEntities: refreshing master data via ODBC")

	if cfg.MICROS3700 == nil {
		log.Printf("[micros3700][WARN] micros3700 config is nil; returning empty snapshot")
		return &entities.Snapshot{}, nil
	}

	db, err := openODBC(cfg)
	if err != nil {
		log.Printf("[micros3700][WARN] openODBC: %v", err)
		return &entities.Snapshot{}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[micros3700][WARN] ODBC ping failed: %v; returning empty snapshot", err)
		return &entities.Snapshot{}, nil
	}

	employees, err := syncEmployeesODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncEmployees: %v", err)
	}

	tables, err := syncTablesODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncTables: %v", err)
	}

	tenders, err := syncTendersODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncTenders: %v", err)
	}

	costCenters, err := syncCostCentersODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncCostCenters: %v", err)
	}

	orderTypes, err := syncOrderTypesODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncOrderTypes: %v", err)
	}

	menuItems, err := syncMenuItemsODBC(db)
	if err != nil {
		log.Printf("[micros3700][WARN] syncMenuItems: %v", err)
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

func syncTicketsODBC(cfg *config.Config) ([]entities.Ticket, error) {
	log.Printf("[micros3700] SyncTickets: querying open checks via ODBC")

	if cfg.MICROS3700 == nil {
		log.Printf("[micros3700][WARN] micros3700 config is nil; returning empty ticket list")
		return []entities.Ticket{}, nil
	}

	db, err := openODBC(cfg)
	if err != nil {
		log.Printf("[micros3700][WARN] openODBC: %v", err)
		return []entities.Ticket{}, nil
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Printf("[micros3700][WARN] ODBC ping failed: %v; returning empty ticket list", err)
		return []entities.Ticket{}, nil
	}

	rows, err := db.Query(`
		SELECT h.chk_seq, h.chk_num, h.rvc_seq, h.emp_seq, h.tab_seq, h.chk_ttl, h.chk_opn_tim
		FROM gst_chk_hdr h
		WHERE h.chk_clsd_flag = 0`)
	if err != nil {
		log.Printf("[micros3700][WARN] query gst_chk_hdr: %v", err)
		return []entities.Ticket{}, nil
	}
	defer rows.Close()

	var tickets []entities.Ticket
	for rows.Next() {
		var chkSeq, chkNum, rvcSeq, empSeq, tabSeq, chkTtl int
		var chkOpnTim time.Time
		if err := rows.Scan(&chkSeq, &chkNum, &rvcSeq, &empSeq, &tabSeq, &chkTtl, &chkOpnTim); err != nil {
			log.Printf("[micros3700][WARN] scan gst_chk_hdr row: %v", err)
			continue
		}
		tickets = append(tickets, entities.Ticket{
			Number:     chkSeq,
			Table:      tabSeq,
			CostCenter: rvcSeq,
			Total:      float64(chkTtl) / 100.0,
			OpenedAt:   chkOpnTim.Format(time.RFC3339),
			Open:       true,
			POSType:    "micros3700",
		})
	}
	if err := rows.Err(); err != nil {
		log.Printf("[micros3700][WARN] rows iteration error: %v", err)
	}
	return tickets, nil
}

func syncEmployeesODBC(db *sql.DB) ([]entities.Employee, error) {
	rows, err := db.Query(`
		SELECT emp_seq, LTRIM(RTRIM(emp_last_name)), LTRIM(RTRIM(emp_first_name))
		FROM emp WHERE emp_active_flag = 1`)
	if err != nil {
		return nil, fmt.Errorf("query emp: %w", err)
	}
	defer rows.Close()

	var out []entities.Employee
	for rows.Next() {
		var empSeq int
		var lastName, firstName string
		if err := rows.Scan(&empSeq, &lastName, &firstName); err != nil {
			return nil, fmt.Errorf("scan emp row: %w", err)
		}
		out = append(out, entities.Employee{
			ID:        strconv.Itoa(empSeq),
			LastName:  lastName,
			FirstName: firstName,
			Store:     "micros3700",
		})
	}
	return out, rows.Err()
}

func syncTablesODBC(db *sql.DB) ([]entities.Table, error) {
	rows, err := db.Query(`
		SELECT obj_num, LTRIM(RTRIM(name)) FROM tab WHERE active_flag = 1`)
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

func syncTendersODBC(db *sql.DB) ([]entities.Tender, error) {
	rows, err := db.Query(`
		SELECT obj_num, LTRIM(RTRIM(name)) FROM tnd WHERE active_flag = 1`)
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

func syncCostCentersODBC(db *sql.DB) ([]entities.CostCenter, error) {
	rows, err := db.Query(`
		SELECT obj_num, LTRIM(RTRIM(name)) FROM rvc WHERE active_flag = 1`)
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

func syncOrderTypesODBC(db *sql.DB) ([]entities.OrderType, error) {
	rows, err := db.Query(`
		SELECT obj_num, LTRIM(RTRIM(name)) FROM ord_type WHERE active_flag = 1`)
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

func syncMenuItemsODBC(db *sql.DB) ([]entities.MenuItem, error) {
	rows, err := db.Query(`
		SELECT mi_seq, LTRIM(RTRIM(name1)), def_price FROM mi_def WHERE active_flag = 1`)
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
