import re

with open('main.go', 'r', encoding='utf-8') as f:
    s = f.read()

# Fix 1: point tablesXML directly at C:\SC\set1.xml
s = s.replace(
    r'tablesXML    = `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export\set1.xml`',
    r'tablesXML    = `C:\SC\set1.xml`'
)

# Fix 2: remove the RunWExportAndCopySet1 call block (3 lines)
s = s.replace(
    '\t\tif err := positouch.RunWExportAndCopySet1(); err != nil {\n\t\t\tlog.Printf("[sync][WARN] WExport failed, tables may be stale: %v", err)\n\t\t}\n\t\t',
    '\t\t'
)

with open('main.go', 'w', encoding='utf-8', newline='\n') as f:
    f.write(s)

print('Done')
