# POSitouch Integration Agent

A pure-Go agent that reads dBASE III/IV (DBF) files produced by POSitouch POS systems, extracts key operational data, caches it locally as JSON, and refreshes every 30 minutes.

## Architecture

```
POSitouch-Integration/
├── main.go          — entry point; parses flags, loads config, starts agent
├── agent/           — refresh loop (30-minute ticker + graceful shutdown)
├── cache/           — thread-safe in-memory store + rooam_cache.json persistence
├── config/          — rooam_config.json loader; derives SC/DBF/ALTDBF paths
├── dbf/             — pure-Go dBASE III/IV reader (no CGO, no external deps)
└── positouch/       — typed readers for each POSitouch DBF file
```

## Data Sources

| File | Location | Contents |
|------|----------|----------|
| `NAMECC.DBF` | DBF dir | Cost centers (primary) |
| `NAMES.DBF` | DBF dir | Cost centers + tenders fallback (CC/PY prefix filtering) |
| `NAMEPAY.DBF` | DBF dir | Tenders / payment types (primary) |
| `USERS.DBF` | DBF dir | Employees |
| `EMPFILE.DBF` | SC dir (fallback: DBF dir) | Employee status enrichment |
| `CHKHDR.DBF` | DBF dir | Tables (primary) |
| `CHECK.DBF` | DBF dir | Tables (fallback) |
| `MENUS.DBF` | SC dir | Menu / order types |

> **Security note:** `SSN` and `SOC_SEC` fields from `EMPFILE.DBF` are intentionally never read or cached.

## Configuration

Create `rooam_config.json`:

```json
{
  "location": {
    "name": "My Restaurant",
    "country": "US",
    "address1": "123 Main St",
    "city": "Springfield",
    "state": "IL",
    "zip": "62701"
  },
  "rooam": {
    "tender_id": "rooam-tender-uuid",
    "employee_id": "rooam-employee-uuid"
  },
  "positouch": {
    "spcwin_path": "C:\\SC\\SPCWIN.ini",
    "virtual_section": "VirtualSection",
    "xml_section": "XMLSection"
  }
}
```

The `spcwin_path` field is used to derive three directories automatically:

| Derived path | Example |
|---|---|
| `SCDir` | `C:\SC` |
| `DBFDir` | `C:\DBF` |
| `AltDBFDir` | `C:\ALTDBF` |

## Building

```powershell
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o rooam-pos-agent.exe
```

## Running

```powershell
rooam-pos-agent.exe -config path\to\rooam_config.json
```

## Output

The agent writes `rooam_cache.json` in the same directory as the config file:

```json
{
  "last_updated": "2026-01-01T12:00:00Z",
  "cost_centers": [{ "store": "01", "code": 1, "name": "Bar" }],
  "tenders":      [{ "store": "01", "code": 1, "name": "Cash" }],
  "employees":    [{ "store": "01", "number": 42, "last_name": "Smith", "first_name": "Jane", "type": 1, "mag_card_id": 0 }],
  "tables":       [{ "store": "01", "number": 10, "cost_center": 1 }],
  "order_types":  [{ "store": "01", "menu_number": 1, "title": "Dine In", "order_type": 0 }]
}
```

## No External Dependencies

The module uses only the Go standard library — `CGO_ENABLED=0` builds are fully supported.
