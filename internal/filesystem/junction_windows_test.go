//go:build windows

package filesystem

import (
	"msk/internal/parser"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestJunctionSafety(t *testing.T) {
	base := t.TempDir()
	target := t.TempDir()
	os.Mkdir(filepath.Join(base, "R"), 0755)
	junction := filepath.Join(base, "R", "linked")
	if b, e := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); e != nil {
		t.Skipf("junction unavailable: %v %s", e, b)
	}
	defer os.Remove(junction)
	n, _ := parser.Parse(`R/{first,linked/{escaped}}`)
	if _, e := Prepare(n, base); e == nil {
		t.Fatal("followed junction")
	}
	if _, e := os.Stat(filepath.Join(base, "R", "first")); !os.IsNotExist(e) {
		t.Fatal("preflight wrote")
	}
	scanned, e := Scan(filepath.Join(base, "R"), ScanOptions{MaxDepth: -1})
	if e != nil || len(scanned.Children) != 0 {
		t.Fatal("scan followed junction", e)
	}
}
