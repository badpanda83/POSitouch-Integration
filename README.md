# Rooam POS Integration Agent (POSitouch + MICROS 3700)

Production-ready Windows service written in Go that bridges on‑premise POS systems with the Rooam cloud server (Railway).

- **POSitouch**: syncs master data + tickets from DBF/XML and supports **order + payment injection** via POSitouch’s XML ordering interface.
- **MICROS RES 3700**: syncs **tickets via RTTP (IFS push over TCP)** and reads **master data via Sybase SQL Anywhere 16 (ODBC)**.

> Repo description says this is a POC, but the codebase now contains a real installer flow and multi‑POS driver support.

---

## Table of Contents

- [Architecture](#architecture)
- [Supported POS Systems](#supported-pos-systems)
- [How It Works](#how-it-works)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
  - [POSitouch config](#positouch-config)
  - [MICROS 3700 config](#micros-3700-config)
- [Building](#building)
- [Installer (MSI)](#installer-msi)
- [Running](#running)
- [Project Structure](#project-structure)
- [Data Sources](#data-sources)
- [Sync Behaviour](#sync-behaviour)
- [Order Processing Flow (POSitouch)](#order-processing-flow-positouch)
- [Payment Processing Flow (POSitouch)](#payment-processing-flow-positouch)
- [Local HTTP API](#local-http-api)
- [MICROS 3700 Notes](#micros-3700-notes)
- [Logging](#logging)
- [Gitignore Notes](#gitignore-notes)
- [Contributing](#contributing)

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│ Windows POS Machine                                                      │
│                                                                          │
│   rooam-pos-agent.exe  ◄──── rooam_config.json                           │
│        │                                                                 │
│        ├── uploads master data to Railway (interval configurable in code)
│        ├── uploads tickets to Railway                                    │
│        ├── polls Railway for pending actions                             │
│        └── exposes local HTTP API (:8080)                                │
│                                                                          │
│  POSitouch mode (pos_type=positouch)                                     │
│    ├── reads DBF + WExport XML                                           │
│    ├── reads tickets from OMNIVORE + OMNIVORE_CLOSE                       │
│    └── writes ORDER*.XML to OMNIVORE_INORDER (orders + payments)          │
│                                                                          │
│  MICROS 3700 mode (pos_type=micros3700)                                  │
│    ├── listens for RTTP pushes on TCP :5454 (default)                     │
│    └── reads master data via Sybase SQL Anywhere 16 through ODBC (Windows)
└──────────────────────────────────────────────────────────────────────────┘
                          │  HTTPS
                          ▼
                 Rooam cloud server (Railway)
```

---

## Supported POS Systems

### POSitouch

- Master data sync from DBF + WExport exports
- Ticket sync from POSitouch XML ticket feeds
- **Order injection** via `ORDER*.XML` drop (POSitouch XML ordering)
- **Payment + close** via `ORDER*.XML` with `Function=4`

### MICROS RES 3700

- **Ticket sync** via IFS/RTTP push interface (TCP). The driver parses RTTP frames and keeps an in‑memory ticket store.
- **Master data sync** via Sybase SQL Anywhere 16 **ODBC** on Windows (32‑bit DSN, default `Micros`).
- **Order creation is not supported** by this driver (RTTP is push‑only).

---

## How It Works

### Startup sequence

1. Load `rooam_config.json`
2. Select driver based on `pos_type` (`positouch` or `micros3700`)
3. Perform initial entity sync
4. Start ticket sync loop
5. Start poller loop for pending actions (POSitouch order/payment injection)
6. Start local HTTP server on `:8080`
7. Wait for SIGINT/SIGTERM

> For POSitouch details (WExport, XML directories, confirmation polling) see the POSitouch flows below.

---

## Prerequisites

| Requirement | Notes |
|---|---|
| Windows machine with the POS installed | Agent should run on the same machine/network segment as the POS system |
| Go | Needed only to build from source |
| Network access to the Railway cloud server | HTTPS outbound (443) |

### POSitouch prerequisites

| Requirement | Notes |
|---|---|
| POSitouch installed | Agent runs on the POSitouch server/workstation |
| `WExport.EXE` available | Typically `C:\SC\WExport.EXE` |
| POSitouch XML ordering enabled | `spcwin.ini` should contain `XMLInOrderPath` and `XMLOutOrderPath` |

### MICROS 3700 prerequisites

| Requirement | Notes |
|---|---|
| MICROS RES 3700 with IFS RTTP enabled | RTTP push should be configured to send to the agent host/port |
| TCP port open | Default **5454** inbound to the agent |
| Sybase SQL Anywhere 16 ODBC DSN | 32‑bit DSN configured on Windows; default DSN name is **`Micros`** |

---

## Configuration

The agent reads `rooam_config.json`.

- For **POSitouch**, you can generate the file using the installer flow or `installer/config_writer`.
- For **MICROS 3700**, there is an example config: `rooam_config.micros3700.example.json`.

### POSitouch config

A typical POSitouch config includes SC/DBF and XML directories:

```json
{
  "pos_type": "positouch",
  "location": { "name": "Smitty's" },
  "install_dir": "C:\Program Files\Rooam\POSAgent",
  "sc_dir": "C:\SC",
  "dbf_dir": "C:\DBF",
  "alt_dbf_dir": "C:\ALTDBF",
  "xml_dir": "C:\SC\XML\OMNIVORE",
  "xml_close_dir": "C:\SC\XML\OMNIVORE_CLOSE",
  "xml_in_order_dir": "C:\SC\XML\OMNIVORE_INORDER",
  "cloud": {
    "enabled": true,
    "endpoint": "https://positouch-cloud-server-production.up.railway.app/api/v1/pos-data",
    "api_key": "your-api-key-here"
  }
}
```

### MICROS 3700 config

Use `pos_type: "micros3700"` and configure RTTP + ODBC:

```json
{
  "pos_type": "micros3700",
  "location": { "name": "My Restaurant", "country": "US", "address1": "456 Elm Street" },
  "rooam": { "tender_id": "rooam-tender-id", "employee_id": "rooam-employee-id" },
  "micros3700": {
    "rttp_port": 5454,
    "odbc_dsn": "Micros",
    "revenue_center_id": 1
  },
  "cloud": {
    "enabled": true,
    "endpoint": "https://your-cloud-server.example.com/api/v1/pos-data",
    "api_key": "YOUR_API_KEY_HERE"
  }
}
```

Notes:

- Legacy config fields like `transaction_services_url`, and older MySQL fields are retained for backward compatibility in `config/config.go`, but the current implementation uses **RTTP + ODBC**.
- ODBC access requires **Windows + CGO** builds (the codebase falls back to an empty snapshot if ODBC is unavailable).

---

## Building

Build the agent binary:

```powershell
go build -ldflags="-w -s" -o rooam-pos-agent.exe .
```

Cross-compile from another OS:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o rooam-pos-agent.exe .
```

---

## Installer (MSI)

There is a full WiX v4 MSI project under `installer/` (see also `installer/README.md`).

Key points:

- The MSI installs the agent as a Windows Service: **`RooamPOSAgent`**.
- The UI includes a **POS Type** choice (**POSitouch** vs **MICROS 3700**) and writes a matching `rooam_config.json` via `config_writer.exe`.
- `installer/detect.ps1` can auto-detect POSitouch paths (`spcwin.exe` + XML directories).

Build the MSI:

```powershell
cd installer
.\build.ps1
```

Output:

```
installer/out/
├── POSitouch-Integration.exe
├── config_writer.exe
└── RooamPOSAgent-Setup.msi
```

---

## Running

```powershell
.\rooam-pos-agent.exe -config .\rooam_config.json
```

To stop: `Ctrl+C` or send `SIGTERM`. The agent shuts down gracefully.

---

## Project Structure

```
POSitouch-Integration/
├── main.go
├── poller.go
├── config/
├── driver/
│   ├── positouch/           # POSitouch driver
│   └── micros3700/          # MICROS 3700 driver (RTTP + ODBC)
├── installer/               # WiX v4 MSI + helpers
├── ordering/
├── positouch/
├── dbf/
├── cache/
└── utils/
```

---

## Data Sources

### POSitouch

(unchanged) DBF and XML sources described below.

### MICROS 3700

- Tickets: RTTP push frames via TCP listener (default 5454)
- Master data: Sybase SQL Anywhere 16 via ODBC DSN (default `Micros`)

---

## Sync Behaviour

- **Tickets**: pushed to the cloud at a regular interval (and/or on driver-specific schedule)
- **Entities**: refreshed periodically and uploaded to the cloud

POSitouch specifics:

### Entity sync (POSitouch)

1. Runs `WExport.EXE ExportSettings <manifest>` to regenerate XML exports
2. Reads all 9 entity types
3. `PUT`s each to `{cloud.endpoint}/{location.name}/{entity}`

### Ticket sync (POSitouch)

1. Calls `positouch.ReadAllTickets(xmlDir, xmlCloseDir)`
2. Deduplicates by ticket number (open wins over closed)
3. `PUT`s array to `{cloud.endpoint}/{location.name}/tickets`

---

## Order Processing Flow (POSitouch)

(Existing diagram/description preserved below — POSitouch only.)

```
Railway                          Agent                         POSitouch
   │                               │                               │
   │  GET /tickets/pending         │                               │
   │◄──────────────────────────────│                               │
   │  [ {ref, payload}, ... ]      │                               │
   │──────────────────────────────►│                               │
   │                               │  WriteOrderXML(req)           │
   │                               │──────────────────────────────►│
   │                               │  ORDER*.XML written           │
   │                               │  to OMNIVORE_INORDER          │
   │                               │                               │
   │                               │  poll OMNIVORE for OUT*.XML   │
   │                               │  every 2s, up to 30s          │
   │                               │◄──────────────────────────────│
   │                               │  OUT*.XML written             │
   │                               │  (ResponseCode=0 = success)   │
   │                               │                               │
   │  PUT /tickets/{ref}/result    │                               │
   │◄──────────────────────────────│                               │
   │  { status:"created",          │                               │
   │    ticket: {...} }            │                               │
```

---

## Payment Processing Flow (POSitouch)

(Existing description preserved below — POSitouch only.)

---

## Local HTTP API

The agent exposes a local HTTP server on `:8080`.

- `POST /api/v1/tickets` — submit a POSitouch order directly (bypasses Railway)
- `GET /health` — liveness check

---

## MICROS 3700 Notes

- RTTP messages are acknowledged with `RECEIVED`.
- The driver keeps tickets in memory for a TTL (currently 4 hours) and returns them from `SyncTickets()`.
- `SyncEntities()` attempts ODBC; if unavailable, it logs a warning and returns an empty snapshot so ticket syncing continues.

---

## Logging

All output goes to stdout. Log prefixes include:

- `[micros3700]` / `[micros3700][rttp]` — MICROS 3700 driver
- `[sync]`, `[ticket_sync]`, `[poller]`, `[server]`, `[positouch]` — shared agent paths

---

## Gitignore Notes

The following files are intentionally excluded from version control:

| File/Pattern | Reason |
|---|---|
| `rooam_config.json` | Contains API keys and local paths — never commit |
| `rooam_cache.json` | Runtime cache — regenerated on startup |
| `rooam-pos-agent.exe` | Built binary — build locally |
| `installer/out/` | Build artifacts |
| `utils/Export/` | WExport output — regenerated on startup |

---

## Contributing

Contributions are welcome. Please open an issue before large changes and update documentation when behaviour changes.

---

*Maintainer: [@badpanda83](https://github.com/badpanda83)*
*Cloud server: [`badpanda83/positouch-cloud-server`](https://github.com/badpanda83/positouch-cloud-server)*
