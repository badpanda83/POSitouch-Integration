@echo off
REM Step 1: Generate latest set1.xml
C:\SC\WExport.EXE ExportSettings C:\Users\Omnivore\Documents\POSitouch-Integration\utils\wexport_layout_manifest.xml
REM Step 2: Copy to your agent's Export folder
copy /Y C:\SC\set1.xml C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export\set1.xml
REM Step 3: Then call your Go agent executable
C:\Users\Omnivore\Documents\POSitouch-Integration\rooam-pos-agent.exe -config .\rooam_config.json