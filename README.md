# POSitouch Integration Agent

A production-ready Windows service written in Go that bridges on-premise [POSitouch](https://www.positouch.com/) POS systems with the [`positouch-cloud-server`](https://github.com/badpanda83/positouch-cloud-server) running on Railway. It reads POS data from DBF and XML files, syncs it to the cloud, processes incoming orders via POSitouch's XML ordering interface, and reports results back to Railway.

---

## Table of Contents

- [Architecture](#architecture)
- [How It Works](#how-it-works)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Building](#building)
- [Running](#running)
- [Project Structure](#project-structure)
- [Data Sources](#data-sources)
- [Sync Behaviour](#sync-behaviour)
- [Order Processing Flow](#order-processing-flow)
- [Local HTTP API](#local-http-api)
- [POSitouch XML Interface](#positouch-xml-interface)
- [Logging](#logging)
- [Gitignore Notes](#gitignore-notes)
- [Contributing](#contributing)

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Windows POS Machine                                          │
│                                                              │
│  POSitouch POS                                               │
│    │                                                         │
│    ├── writes DBF files (C:\DBF\, C:\SC\)                   │
│    ├── reads  ORDER*.XML  (OMNIVORE_INORDER dir)             │
│    └── writes OUT*.XML    (OMNIVORE dir — confirmations)     │
│                                                              │
│  rooam-pos-agent.exe  ◄──── rooam_config.json               │
│    │                                                         │
│    ├── reads DBF/XML → syncs to Railway every 30min          │
│    ├── syncs tickets to Railway every 30s                    │
│    ├── polls Railway for pending orders every 5s             │
│    ├── writes ORDER*.XML to OMNIVORE_INORDER                 │
│    ├── polls OMNIVORE for OUT*.XML confirmation (≤30s)       │
│    └── PUTs result back to Railway                           │
│                                                              │
│  :8080 (local HTTP API)                                      │
│    └── POST /api/v1/tickets  (direct local order endpoint)   │
└──────────────────────────────────────────────────────────────┘
                         │  HTTPS
                         ▼
         positouch-cloud-server (Railway)
```

---

## How It Works

### Startup sequence

1. Kill any stale `WExport.EXE` processes
2. Load `rooam_config.json`
3. Run `WExport.EXE` to regenerate `set1.xml` (table layout)
4. Read all entities from DBF/XML files
5. Upload all entities to Railway cloud server
6. Start ticket sync goroutine (every 30 seconds)
7. Start entity sync goroutine (every 30 minutes)
8. Start order poller goroutine (every 5 seconds)
9. Start local HTTP server on `:8080`
10. Wait for SIGINT/SIGTERM

### On every tick

- **Every 30s:** Read all open+closed tickets from `C:\SC\XML\OMNIVORE\` and `C:\SC\XML\OMNIVORE_CLOSE\` and `PUT` them to Railway.
- **Every 30min:** Re-run WExport, re-read all DBF/XML files, re-upload all 9 entity types.
- **Every 5s:** `GET` pending orders from Railway, process each one (see below).

---

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| Windows machine with POSitouch installed | Agent must run on the same machine as the POS |
| Go 1.21+ | For building only — the binary has no runtime deps |
| `WExport.EXE` | Must be present at `C:\SC\WExport.EXE` |
| POSitouch XML ordering enabled | `spcwin.ini` must have `XMLInOrderPath` and `XMLOutOrderPath` configured |
| Network access to Railway | HTTPS outbound on port 443 |

---

## Configuration

Create `rooam_config.json` in the same directory as the binary. **This file is gitignored and must never be committed.**

```json
{
  "location": {
    "name": "Smitty's"
  },
  "install_dir": "C:\\Users\\Omnivore\\Documents\\POSitouch-Integration",
  "sc_dir": "C:\\SC",
  "dbf_dir": "C:\\DBF",
  "alt_dbf_dir": "C:\\ALTDBF",
  "xml_dir": "C:\\SC\\XML\\OMNIVORE",
  "xml_close_dir": "C:\\SC\\XML\\OMNIVORE_CLOSE",
  "xml_in_order_dir": "C:\\SC\\XML\\OMNIVORE_INORDER",
  "cloud": {
    "endpoint": "https://positouch-cloud-server-production.up.railway.app/api/v1/pos-data",
    "api_key": "your-api-key-here"
  }
}
```

### Configuration fields

| Field             | Description                                                    |
|-------------------|----------------------------------------------------------------|
| `location.name`   | Location identifier — must match the `locationID` used by Rooam |
| `install_dir`     | Directory containing the agent binary and config               |
| `sc_dir`          | POSitouch SC directory (contains `WExport.EXE`, `SPCWIN.ini`) |
| `dbf_dir`         | Directory containing DBF data files                            |
| `alt_dbf_dir`     | Alternate DBF directory (fallback for some files)              |
| `xml_dir`         | POSitouch OMNIVORE directory — contains open tickets + OUT*.XML confirmations |
| `xml_close_dir`   | POSitouch OMNIVORE_CLOSE directory — contains closed tickets   |
| `xml_in_order_dir`| POSitouch OMNIVORE_INORDER directory — agent writes ORDER*.XML here |
| `cloud.endpoint`  | Railway cloud server base URL                                  |
| `cloud.api_key`   | Bearer token sent with all Railway requests                    |

> ⚠️ **`xml_dir` must be `OMNIVORE` not `OMNIVORE_OPEN`.** The confirmation files (`OUT*.XML`) are written to `OMNIVORE` by POSitouch.

---

## Building

```powershell
cd C:\Users\Omnivore\Documents\POSitouch-Integration
go build -ldflags="-w -s" -o rooam-pos-agent.exe .
```

Cross-compile from another OS:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o rooam-pos-agent.exe .
```

---

## Running

```powershell
.\rooam-pos-agent.exe -config .\rooam_config.json
```

To stop: `Ctrl+C` or send `SIGTERM`. The agent shuts down gracefully.

**Startup log (expected output):**

```
╔══════════════════════════════════════════╗
║  rooam-pos-agent v1.0.0                  ║
║  POSitouch Integration Agent             ║
╚══════════════════════════════════════════╝

[main] config path : .\rooam_config.json
[main] location    : Smitty's
[main] SC dir      : C:\SC\
[main] DBF dir     : C:\DBF\
[sync] WExport completed, set1.xml refreshed
[sync] Loaded 1075 menu items ...
[sync] uploaded employees, response status: 200 OK
...
[sync] All entities uploaded for location: Smitty's
[ticket_sync] polling tickets every 30s
[agent] polling Railway for pending orders every 5s
[server] starting API on :8080
```

---

## Project Structure

```
POSitouch-Integration/
├── main.go              # Entry point — config, startup, goroutines, HTTP server
├── poller.go            # pollPendingOrders, reportOrderResult, putOrderResult
├── config/
│   └── config.go        # rooam_config.json loader
├── ordering/
│   └── ordering.go      # CreateTicket HTTP handler, WriteOrderXML, FindConfirmation
├── positouch/
│   ├── ticket_cache.go  # ReadAllTickets — parses OMNIVORE + OMNIVORE_CLOSE XML
│   ├── menu.go          # ParseMenuXML, ParseMenuCategories, ParseMenuModifiers
│   ├── tables.go        # ParseTablesFromSet1XML
│   ├── employees.go     # ReadEmployees (USERS.DBF + EMPFILE.DBF)
│   ├── tenders.go       # ReadTenders (NAMEPAY.DBF / NAMES.DBF fallback)
│   ├── cost_centers.go  # ReadCostCenters (NAMECC.DBF)
│   ├── order_types.go   # ReadOrderTypes (MENUS.DBF)
│   └── order.go         # XML structs for outbound ORDER*.XML
├── dbf/
│   └── reader.go        # Pure-Go dBASE III/IV file reader (no CGO)
├── cache/
│   └── cache.go         # Thread-safe in-memory cache + Data struct
├── utils/
│   ├── wexport_layout_manifest.xml   # WExport manifest
│   └── Export/                       # Generated by WExport (gitignored)
├── rooam_config.json    # Local config (gitignored)
└── rooam-pos-agent.exe  # Built binary (gitignored)
```

---

## Data Sources

| Entity        | Primary source                        | Fallback                |
|---------------|---------------------------------------|-------------------------|
| Cost Centers  | `C:\DBF\NAMECC.DBF`                   | `NAMES.DBF` (CC prefix) |
| Tenders       | `C:\DBF\NAMEPAY.DBF`                  | `NAMES.DBF` (PY prefix) |
| Employees     | `C:\DBF\USERS.DBF`                    | + `EMPFILE.DBF` if present |
| Tables        | `utils/Export/set1.xml` (via WExport) | —                       |
| Order Types   | `C:\DBF\MENUS.DBF`                    | —                       |
| Open Tickets  | `C:\SC\XML\OMNIVORE\*.XML`            | —                       |
| Closed Tickets| `C:\SC\XML\OMNIVORE_CLOSE\*.XML`      | —                       |
| Menu Items    | `utils/Export/menu_items.xml`         | —                       |
| Categories    | `utils/Export/menu_categories.xml`    | —                       |
| Modifiers     | `utils/Export/menu_items.xml` (MajorCategory=11) | —          |

> **Security:** `SSN` and `SOC_SEC` fields from `EMPFILE.DBF` are never read or uploaded.

---

## Sync Behaviour

### Entity sync (every 30 minutes + startup)

1. Runs `WExport.EXE ExportSettings <manifest>` to regenerate XML exports
2. Reads all 9 entity types
3. `PUT`s each to `{cloud.endpoint}/{location.name}/{entity}`
4. Logs count and HTTP status for each

### Ticket sync (every 30 seconds)

1. Calls `positouch.ReadAllTickets(xmlDir, xmlCloseDir)`
2. Deduplicates by ticket number (open wins over closed)
3. `PUT`s array to `{cloud.endpoint}/{location.name}/tickets`

---

## Order Processing Flow

```
Railway                          Agent                         POSitouch
   │                               │                               │
   │  GET /tickets/pending         │                               │
   │◄──────────────────────────────│                               │
   │  [ {ref, payload}, ... ]      │                               │
   │──────────────────────────────►│                               │
   │                               │  WriteOrderXML(req)           │
   │                               │──────────────────────────────►│
   │                               │  ORDER        .XML written    │
   │                               │  to OMNIVORE_INORDER          │
   │                               │                               │
   │                               │  poll OMNIVORE for OUT*.XML   │
   │                               │  every 2s, up to 30s         │
   │                               │◄──────────────────────────────│
   │                               │  OUT   .XML written           │
   │                               │  (ResponseCode=0 = success)   │
   │                               │                               │
   │  PUT /tickets/{ref}/result    │                               │
   │◄──────────────────────────────│                               │
   │  { status:"created",          │                               │
   │    ticket: {...} }            │                               │
```

### On each poll cycle (every 5s)

For each pending order:
1. Unmarshal `payload` into `CreateTicketRequest`
2. Call `ordering.WriteOrderXML(req, xmlInOrderDir)` — atomically writes `ORDER<random>.XML`
3. Launch `go reportOrderResult(...)` — non-blocking, so other orders can be processed immediately
4. If `WriteOrderXML` fails: immediately `PUT` `status="failed"` to Railway

### In `reportOrderResult`

1. Poll `xmlDir` every 2 seconds for `OUT*.XML` where `<ReferenceNumber>` matches
2. On finding confirmation file:
   - Delete the `OUT*.XML` file
   - If `ResponseCode == 0`: read tickets, find matching ticket by `table == req.TableNumber && opened within 60s`, `PUT` `status="created"` with full ticket
   - If `ResponseCode != 0`: `PUT` `status="failed"` with error text from `<Error><Text>`
3. After 30 seconds with no confirmation: `PUT` `status="failed"` with timeout message

---

## Local HTTP API

The agent exposes a local HTTP server on `:8080`. This is useful for direct testing on the POS machine but is **not reachable from outside** (no port forwarding required or recommended).

### `POST /api/v1/tickets`

Submit an order directly to the agent (bypasses Railway). The connection is held open synchronously — same logic as the cloud path but without the Railway hop.

**Request body** — same format as `POST /api/v1/pos-data/{locationID}/tickets` on Railway (see cloud server README).

**Responses:**

| Status | Meaning |
|--------|---------|
| `201 Created` | POSitouch confirmed — body contains `{"status":"created","ticket":{...}}` |
| `400 Bad Request` | Validation failure or POSitouch rejection |
| `504 Gateway Timeout` | No `OUT*.XML` received within 30s |

### `GET /api/v1/pos-data/{locationID}/{entity}`

Read the locally cached entity data (same data that was last uploaded to Railway).

### `GET /health`

```json
{ "status": "ok" }
```

---

## POSitouch XML Interface

### Outbound order file (`ORDER*.XML`)

Written to `xml_in_order_dir`. POSitouch picks this up automatically.

```xml

```

### Inbound confirmation file (`OUT*.XML`)

Written by POSitouch to `xml_dir` after processing an order.

```xml

```

The agent deletes the `OUT*.XML` file after reading it.

---

## Logging

All output goes to stdout. Log prefixes:

| Prefix | Source |
|--------|--------|
| `[main]` | Startup and config |
| `[sync]` | Entity sync (30min cycle) |
| `[ticket_sync]` | Ticket sync (30s cycle) |
| `[ticket_cache]` | Ticket XML parsing |
| `[WExport]` | WExport.EXE execution |
| `[poller]` | Pending order polling and result reporting |
| `[orders]` | Local HTTP order handler |
| `[server]` | Local HTTP server |
| `[positouch]` | DBF file reading |

---

## Gitignore Notes

The following files are intentionally excluded from version control:

| File/Pattern | Reason |
|---|---|
| `rooam_config.json` | Contains API keys and local paths — never commit |
| `rooam_cache.json` | Runtime cache — regenerated on startup |
| `rooam-pos-agent.exe` | Built binary — build locally |
| `utils/Export/` | WExport output — regenerated on startup |
| `*.LOG`, `agent*.log` | Runtime logs |

---

## Contributing

Contributions are welcome. Please open an issue before large changes. All PRs should:

- Follow standard Go formatting (`gofmt`).
- Include tests for new functionality.
- Update this README if behaviour changes.

---

*Maintainer: [@badpanda83](https://github.com/badpanda83)*
*Cloud server: [`badpanda83/positouch-cloud-server`](https://github.com/badpanda83/positouch-cloud-server)*
