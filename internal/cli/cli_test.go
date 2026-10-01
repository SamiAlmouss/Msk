package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeClip struct {
	text string
	err  error
}

func (f *fakeClip) Read() (string, error) { return f.text, f.err }
func (f *fakeClip) Write(s string) error {
	if f.err != nil {
		return f.err
	}
	f.text = s
	return nil
}
func run(args []string, c *fakeClip) (int, string, string) {
	var out, err bytes.Buffer
	code := (App{Clipboard: c, Out: &out, Err: &err, Version: "test"}).Run(args)
	return code, out.String(), err.String()
}
func TestClipboardAndTreeFile(t *testing.T) {
	base := t.TempDir()
	c := &fakeClip{text: `R/{file,empty/}`}
	code, out, err := run([]string{"--output", base}, c)
	if code != 0 || !strings.Contains(out, "Created files: 1") || err != "" {
		t.Fatal(code, out, err)
	}
	source := filepath.Join(base, "tree.txt")
	os.WriteFile(source, []byte("FileRoot/\n└── LICENSE"), 0644)
	for _, args := range [][]string{{source, "--preview", "--output", base}, {"--preview", source, "--output", base}} {
		code, _, err = run(args, c)
		if code != 0 {
			t.Fatal(err)
		}
	}
	if _, e := os.Stat(filepath.Join(base, "FileRoot")); !os.IsNotExist(e) {
		t.Fatal("preview wrote")
	}
	code, _, err = run([]string{source, "--output", base}, c)
	if code != 0 {
		t.Fatal(err)
	}
}
func TestOutputSeparationAndSaveExclusion(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "tree.txt")
	os.WriteFile(source, []byte("R/\n└── empty/"), 0644)
	save := filepath.Join(base, "compact.txt")
	c := &fakeClip{}
	code, out, err := run([]string{"convert", "--compact", source, "--save", save, "--copy"}, c)
	if code != 0 || out != `R/{empty/}` || !strings.Contains(err, "Saved:") || c.text != out {
		t.Fatal(code, out, err)
	}
	code, _, _ = run([]string{"convert", source, "--compact", "--save", save}, c)
	if code != 1 {
		t.Fatal("overwrite allowed")
	}
	code, out, err = run([]string{"scan", base, "--compact", "--save", save, "--overwrite-output", "--max-depth", "2"}, c)
	if code != 0 || strings.Contains(out, "compact.txt") || !strings.Contains(err, "limited") {
		t.Fatal(code, out, err)
	}
}
func TestErrors(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"scan", ".", "--tree", "--compact"}, {"convert", "a"}, {"a", "b"}, {"--preview=true"}, {"scan"}, {"scan", ".", "--max-depth", "-1"}, {"convert", "--clipboard", "a", "--tree"}, {"--overwrite-output"}, {"scan", ".", "--output", "x"}, {"convert", "a", "--tree", "--exclude", "x"}, {"--output"}} {
		code, out, err := run(args, &fakeClip{})
		if code != 2 || out != "" || err == "" {
			t.Fatal(args, code, out, err)
		}
	}
	code, _, _ := run(nil, &fakeClip{text: `R/{..}`})
	if code != 2 {
		t.Fatal(code)
	}
	code, _, _ = run(nil, &fakeClip{err: fmt.Errorf("busy")})
	if code != 1 {
		t.Fatal(code)
	}
	code, _, _ = run([]string{"missing.txt"}, &fakeClip{})
	if code != 1 {
		t.Fatal(code)
	}
}
func TestDoubleDash(t *testing.T) {
	base := t.TempDir()
	source := filepath.Join(base, "-tree.txt")
	os.WriteFile(source, []byte("R/"), 0644)
	code, _, err := run([]string{"--output", base, "--preview", "--", source}, &fakeClip{})
	if code != 0 {
		t.Fatal(err)
	}
}
func TestInvalidNoWrite(t *testing.T) {
	base := t.TempDir()
	code, _, _ := run([]string{"--output", base}, &fakeClip{text: `R/{first,../escape}`})
	if code != 2 {
		t.Fatal(code)
	}
	entries, _ := os.ReadDir(base)
	if len(entries) != 0 {
		t.Fatal("invalid input wrote")
	}
}
func TestHelpVersion(t *testing.T) {
	for _, arg := range []string{"--help", "--version"} {
		code, out, err := run([]string{arg}, nil)
		if code != 0 || out == "" || err != "" {
			t.Fatal(code, out, err)
		}
	}
}
