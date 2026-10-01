//go:build !windows

package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanAccessFailure(t *testing.T) {
	base := t.TempDir()
	denied := filepath.Join(base, "denied")
	if e := os.Mkdir(denied, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(denied, 0); e != nil {
		t.Fatal(e)
	}
	defer os.Chmod(denied, 0700)
	if _, e := os.ReadDir(denied); e == nil {
		t.Skip("environment bypasses file permissions")
	}
	n, e := Scan(base, ScanOptions{MaxDepth: -1})
	if e == nil || n != nil || !strings.Contains(e.Error(), denied) {
		t.Fatalf("expected path-bearing failure, got %v %v", n, e)
	}
}
