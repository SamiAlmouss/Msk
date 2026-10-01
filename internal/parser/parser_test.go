package parser_test

import (
	"msk/internal/formatter"
	"msk/internal/model"
	"msk/internal/parser"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestTVSnake(t *testing.T) {
	a, e := os.ReadFile("../../examples/tree.txt")
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("../../examples/compact.txt")
	if e != nil {
		t.Fatal(e)
	}
	x, e := parser.Parse(string(a))
	if e != nil {
		t.Fatal(e)
	}
	y, e := parser.Parse(string(b))
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(x, y) {
		t.Fatal("example models differ")
	}
}
func TestRoundTrips(t *testing.T) {
	cases := []string{
		`Root/{Dockerfile,LICENSE,empty/,src/a.go,src/b.go,"مجلد عربي"/{"notes, draft.txt","a{b}.txt"}," leading"/}`,
		"\ufeffRoot/\r\n├── src/a.go\r\n├── src/b.go\r\n└── empty/\r\n",
		"Root\n└── dir\n    └── file\n",
		"Root/\n├── \"├── name\"/\n└── \"escaped\\u0020name\"\n",
		"```text\nRoot/\n└── empty/\n```\n",
		`"My Project"/{"source files"/{main.go},"notes, draft.txt",empty/}`,
		`Root/{"\u0645\u0644\u0641", "tab\u0020name"}`,
		`EmptyProject/`,
		`R/{a/{b/{c/{d}}}}`,
		"Root/\n\n└── a#comment\n",
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			n, e := parser.Parse(s)
			if e != nil {
				t.Fatal(e)
			}
			for _, out := range []string{formatter.Tree(n), formatter.Compact(n)} {
				m, e := parser.Parse(out)
				if e != nil {
					t.Fatalf("%q: %v", out, e)
				}
				if !reflect.DeepEqual(n, m) {
					t.Fatalf("round trip changed model: %q", out)
				}
				if strings.Contains(formatter.Compact(n), "\n") {
					t.Fatal("compact has newline")
				}
				if formatter.Tree(n) != formatter.Tree(m) {
					t.Fatal("unstable tree")
				}
			}
		})
	}
}
func TestInvalid(t *testing.T) {
	for _, s := range []string{"", " \n", `R/{a`, `R/{a}}`, `R/{a,}`, `R/{,a}`, `R/{a,,b}`, `R/{"oops}`, `R/{"bad\q"}`, `R/{"a/b"}`, `R/{"a\\b"}`, `R/{../x}`, `C:/x/`, `//server/a/`, `R/{a//b}`, `R/{CON.txt}`, `R/{COM¹}`, `R/{NUL}`, `R/{"trailing "}`, `R/{x.}`, `R/{a:a}`, `R/{a,a}`, `R/{a,A}`, `R/{a,a/}`, `R/{a/,a}`, `R/{"\u0000"}`, `R/{x?}`, "R/\n        └── a", "R/\nother/", "```text\nR/\n```\nextra", "R/\n└── \"a\"junk"} {
		t.Run(s, func(t *testing.T) {
			if _, e := parser.Parse(s); e == nil {
				t.Fatalf("accepted invalid %q", s)
			}
		})
	}
}
func TestMergeOrder(t *testing.T) {
	n, e := parser.Parse(`R/{src/a,other,src/b,src/}`)
	if e != nil {
		t.Fatal(e)
	}
	if formatter.Compact(n) != `R/{src/{a,b},other}` {
		t.Fatal(formatter.Compact(n))
	}
}
func TestDepthLimit(t *testing.T) {
	if _, e := parser.Parse("R/" + strings.Repeat("{a/", 300) + strings.Repeat("}", 300)); e == nil {
		t.Fatal("unbounded depth")
	}
}
func FuzzParsers(f *testing.F) {
	for _, s := range []string{`R/{a,empty/}`, "R/\n└── a", `"عربي"/{"a,b"}`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		for _, parse := range []func(string) (*model.Node, error){parser.Tree, parser.Compact, parser.Parse} {
			n, e := parse(s)
			if e != nil {
				continue
			}
			for _, output := range []string{formatter.Tree(n), formatter.Compact(n)} {
				m, e := parser.Parse(output)
				if e != nil || !reflect.DeepEqual(n, m) {
					t.Fatalf("round trip failed for %q: %v", output, e)
				}
			}
		}
	})
}
