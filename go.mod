module github.com/badpanda83/POSitouch-Integration

go 1.25.0

require (
	github.com/badpanda83/POSitouch-Integration/positouch v0.1.0
	github.com/go-sql-driver/mysql v1.8.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/badpanda83/POSitouch-Integration/dbf v0.1.0 // indirect
)

replace github.com/badpanda83/POSitouch-Integration/positouch => ./positouch

replace github.com/badpanda83/POSitouch-Integration/dbf => ./dbf
