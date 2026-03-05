# Rooam POS Agent — Windows Installer

This directory contains all artifacts needed to build a proper Windows `.msi` installer for the Rooam POS Integration Agent.

---

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.21+ | <https://go.dev/dl/> |
| WiX Toolset v4 | 4.x | `dotnet tool install --global wix` |
| PowerShell | 5.1+ | Included in Windows 10/11 |
| Windows SDK | optional | Only required if you want to sign the MSI |

> **Cross-compile note**: `build.ps1` can be run from **any** OS that has Go installed because it sets `GOOS=windows GOARCH=amd64` before calling `go build`. WiX itself must run on Windows (or inside a Windows CI runner).

---

## Build steps

Open a PowerShell prompt in the **`installer/`** directory and run:

```powershell
.\build.ps1
```

The script will:
1. Cross-compile `POSitouch-Integration.exe` (the agent binary) from the repository root.
2. Run `wix build` to produce the MSI from `installer.wxs`.
3. Report success or failure with clear console output.

---

## Output

```
installer/out/
├── POSitouch-Integration.exe   ← compiled agent binary
└── RooamPOSAgent-Setup.msi     ← the final installer
```

---

## Testing the installer locally

1. Copy `installer/out/RooamPOSAgent-Setup.msi` to a Windows machine (or VM).
2. Double-click the `.msi`, or run silently:

   ```cmd
   msiexec /i RooamPOSAgent-Setup.msi /qn
   ```

3. After installation, verify the service was registered:

   ```powershell
   Get-Service -Name RooamPOSAgent
   ```

4. Check the health endpoint (service must be running):

   ```powershell
   Invoke-RestMethod http://localhost:8080/health
   ```

   Expected response: `{ "status": "ok" }`

5. Review the written config file at:

   ```
   C:\Program Files\Rooam\POSAgent\rooam_config.json
   ```

### Re-detecting POSitouch paths

If you need to discover the `spcwin.exe` location and XML directories before running the installer UI, run the detection script separately:

```powershell
# Auto-detect from default location
.\detect.ps1

# Provide a hint path
.\detect.ps1 -SpcwinHint "D:\POS\SC\spcwin.exe"
```

### Writing the config file manually

The `config_writer` helper lets you generate `rooam_config.json` without the full MSI install flow:

```powershell
# Build the helper first (from repo root)
go build -o installer/out/config_writer.exe ./installer/config_writer

# Run it
.\out\config_writer.exe `
  -location-name "My Restaurant" `
  -address "123 Main St" `
  -phone "555-1234" `
  -email "pos@myrestaurant.com" `
  -employee-id "EMP001" `
  -tender-id "TENDER01" `
  -api-key "my-secret-api-key" `
  -spcwin-path 'C:\SC\spcwin.exe' `
  -xml-dir 'C:\SC\XML' `
  -xml-close-dir 'C:\SC\XMLCLOSE' `
  -xml-inorder-dir 'C:\SC\XMLIN' `
  -output 'C:\Program Files\Rooam\POSAgent\rooam_config.json'
```

---

## Directory layout

```
installer/
├── README.md              ← this file
├── build.ps1              ← build script (Go → EXE → MSI)
├── detect.ps1             ← POSitouch path auto-detection
├── installer.wxs          ← WiX v4 MSI definition
├── config_writer/
│   └── main.go            ← standalone config-writer CLI
└── out/                   ← build artifacts (git-ignored)
    ├── POSitouch-Integration.exe
    └── RooamPOSAgent-Setup.msi
```
