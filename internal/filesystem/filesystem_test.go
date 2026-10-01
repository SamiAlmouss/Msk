package filesystem

import (
	"bytes"
	"msk/internal/parser"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCreateAndPreserve(t *testing.T) {
	base := t.TempDir()
	n, e := parser.Parse(`R/{empty/,src/{a,b},LICENSE}`)
	if e != nil {
		t.Fatal(e)
	}
	p, e := Prepare(n, base)
	if e != nil {
		t.Fatal(e)
	}
	var preview bytes.Buffer
	p.Preview(&preview)
	if _, e = os.Stat(p.Destination); !os.IsNotExist(e) {
		t.Fatal("preview wrote")
	}
	r, e := p.Execute()
	if e != nil {
		t.Fatal(e)
	}
	if r.Files != 3 || r.Directories != 3 {
		t.Fatalf("%+v", r)
	}
	a := filepath.Join(base, "R", "src", "a")
	if e = os.WriteFile(a, []byte("keep me"), 0644); e != nil {
		t.Fatal(e)
	}
	p, e = Prepare(n, base)
	if e != nil {
		t.Fatal(e)
	}
	r, e = p.Execute()
	if e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(a)
	if string(data) != "keep me" || r.Skipped != 6 {
		t.Fatalf("preservation failed: %+v", r)
	}
	scanned, e := Scan(p.Destination, ScanOptions{MaxDepth: -1})
	if e != nil {
		t.Fatal(e)
	}
	if len(scanned.Children) != 3 {
		t.Fatal(scanned)
	}
}
func TestPreflightNoWrites(t *testing.T) {
	base := t.TempDir()
	os.Mkdir(filepath.Join(base, "R"), 0755)
	os.Mkdir(filepath.Join(base, "R", "clash"), 0755)
	n, _ := parser.Parse(`R/{first,clash}`)
	if _, e := Prepare(n, base); e == nil {
		t.Fatal("accepted clash")
	}
	if _, e := os.Stat(filepath.Join(base, "R", "first")); !os.IsNotExist(e) {
		t.Fatal("preflight wrote")
	}
}
func TestConcurrentConflict(t *testing.T) {
	base := t.TempDir()
	n, _ := parser.Parse(`R/{first,clash}`)
	p, e := Prepare(n, base)
	if e != nil {
		t.Fatal(e)
	}
	os.Mkdir(p.Destination, 0755)
	os.Mkdir(filepath.Join(p.Destination, "clash"), 0755)
	r, e := p.Execute()
	if e == nil || len(r.Created) != 0 {
		t.Fatal("second preflight failed")
	}
}
func TestScanOptionsAndSave(t *testing.T) {
	base := t.TempDir()
	os.MkdirAll(filepath.Join(base, "z", "deep"), 0755)
	os.Mkdir(filepath.Join(base, "empty"), 0755)
	for _, s := range []string{"LICENSE", ".hidden", "tree.txt", "عربي"} {
		os.WriteFile(filepath.Join(base, s), nil, 0644)
	}
	out := filepath.Join(base, "tree.txt")
	n, e := Scan(base, ScanOptions{MaxDepth: 1, Exclude: []string{".hidden"}, SavePath: out})
	if e != nil {
		t.Fatal(e)
	}
	var names []string
	for _, c := range n.Children {
		names = append(names, c.Name)
		if c.Dir && len(c.Children) != 0 {
			t.Fatal("depth ignored")
		}
	}
	if !reflect.DeepEqual(names, []string{"empty", "z", "LICENSE", "عربي"}) {
		t.Fatal(names)
	}
	n, e = Scan(base, ScanOptions{MaxDepth: 0})
	if e != nil || len(n.Children) != 0 {
		t.Fatal("root depth", e)
	}
	n, e = Scan(base, ScanOptions{MaxDepth: -1})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, c := range n.Children {
		if c.Name == ".hidden" {
			found = true
		}
	}
	if !found {
		t.Fatal("hidden skipped")
	}
	if e = Save(out, "new", false); e == nil {
		t.Fatal("overwrote output")
	}
	if e = Save(out, "new", true); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(out)
	if string(b) != "new" {
		t.Fatal("save failed")
	}
	if e = Save(filepath.Join(base, "fresh"), "fresh", false); e != nil {
		t.Fatal(e)
	}
}
func TestSymlinkSafety(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	os.Mkdir(filepath.Join(base, "R"), 0755)
	link := filepath.Join(base, "R", "link")
	if e := os.Symlink(outside, link); e != nil {
		t.Skipf("symlink unavailable: %v", e)
	}
	n, _ := parser.Parse(`R/{new,link/{escaped}}`)
	if _, e := Prepare(n, base); e == nil {
		t.Fatal("followed symlink")
	}
	var w bytes.Buffer
	scanned, e := Scan(filepath.Join(base, "R"), ScanOptions{MaxDepth: -1, Warnings: &w})
	if e != nil || len(scanned.Children) != 0 || !strings.Contains(w.String(), "Skipping link") {
		t.Fatal(e, w.String())
	}
	if _, e = Scan(link, ScanOptions{MaxDepth: -1}); e == nil {
		t.Fatal("scanned link root")
	}
	if e = Save(filepath.Join(link, "output"), "bad", false); e == nil {
		t.Fatal("saved through link")
	}
}
func TestMissingOutputParents(t *testing.T) {
	base := filepath.Join(t.TempDir(), "new", "output")
	n, _ := parser.Parse("R/")
	p, e := Prepare(n, base)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Execute(); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(filepath.Join(base, "R")); e != nil {
		t.Fatal(e)
	}
}
