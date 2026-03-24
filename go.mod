module github.com/badpanda83/POSitouch-Integration

go 1.20

require (
	github.com/alexbrainman/odbc v0.0.0-20250601004241-49e6b2bc0cf0
	github.com/badpanda83/POSitouch-Integration/positouch v0.1.0
	golang.org/x/sys v0.18.0
)

require github.com/badpanda83/POSitouch-Integration/dbf v0.1.0 // indirect

replace github.com/badpanda83/POSitouch-Integration/positouch => ./positouch

replace github.com/badpanda83/POSitouch-Integration/dbf => ./dbf
