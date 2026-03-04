package positouch

import (
"bytes"
"context"
"io"
"log"
"os"
"os/exec"
"sync"
"time"
)

const (
WExportEXE      = `C:\SC\WExport.EXE`
WExportDir      = `C:\SC\`
WExportManifest = `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\wexport_layout_manifest.xml`
Set1XMLSrc      = `C:\SC\set1.xml`
Set1XMLDst      = `C:\Users\Omnivore\Documents\POSitouch-Integration\utils\Export\set1.xml`
)

var wexportMu sync.Mutex

// RunWExportAndCopySet1 runs WExport.exe to regenerate set1.xml then copies
// it to the Export folder so no other process can overwrite it before we read it.
// A package-level mutex ensures only one WExport.exe runs at a time; concurrent
// callers block until the current run finishes. A 60-second context timeout
// automatically kills a hung WExport.exe so the mutex is never held indefinitely.
func RunWExportAndCopySet1() error {
wexportMu.Lock()
defer wexportMu.Unlock()

ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, WExportEXE, "ExportSettings", WExportManifest)
cmd.Dir = WExportDir
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

log.Printf("[WExport] Running: %v", cmd.Args)
err := cmd.Run()
log.Printf("[WExport] STDOUT: %s", stdout.String())
if stderr.Len() > 0 {
log.Printf("[WExport] STDERR: %s", stderr.String())
}
if err != nil {
log.Printf("[WExport][ERROR] %v", err)
return err
}
log.Printf("[WExport] Export completed successfully")

in, err := os.Open(Set1XMLSrc)
if err != nil {
log.Printf("[WExport][ERROR] reading %s: %v", Set1XMLSrc, err)
return err
}
defer in.Close()

out, err := os.Create(Set1XMLDst)
if err != nil {
log.Printf("[WExport][ERROR] creating %s: %v", Set1XMLDst, err)
return err
}
defer out.Close()

if _, err = io.Copy(out, in); err != nil {
log.Printf("[WExport][ERROR] copying set1.xml: %v", err)
return err
}
log.Printf("[WExport] set1.xml copied to %s", Set1XMLDst)
return nil
}
