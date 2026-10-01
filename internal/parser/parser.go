// Package parser parses documents without performing filesystem operations.
package parser

import (
	"encoding/json"
	"fmt"
	"msk/internal/model"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxDepth = 256
const MaxInput = 16 << 20

func prepare(s string) (string, error) {
	if len(s) > MaxInput {
		return "", fmt.Errorf("input exceeds 16 MiB limit")
	}
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("input is not UTF-8")
	}
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	a, b := 0, len(lines)
	for a < b && strings.TrimSpace(lines[a]) == "" {
		a++
	}
	for b > a && strings.TrimSpace(lines[b-1]) == "" {
		b--
	}
	if a == b {
		return "", fmt.Errorf("input is empty")
	}
	if lines[a] == "```" || lines[a] == "```text" {
		if b-a < 3 || lines[b-1] != "```" {
			return "", fmt.Errorf("incomplete outer Markdown fence")
		}
		s = strings.Join(lines[a+1:b-1], "\n")
	}
	return s, nil
}

func Parse(s string) (*model.Node, error) {
	s, e := prepare(s)
	if e != nil {
		return nil, e
	}
	for _, line := range strings.Split(s, "\n") {
		prefix := strings.TrimLeft(line, " │")
		if strings.HasPrefix(prefix, "├── ") || strings.HasPrefix(prefix, "└── ") {
			return tree(s)
		}
	}
	quoted, escaped := false, false
	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if quoted && r == '\\' {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
			continue
		}
		if !quoted && strings.ContainsRune("{},", r) {
			return compact(s)
		}
	}
	return tree(s)
}
func Tree(s string) (*model.Node, error) {
	s, e := prepare(s)
	if e != nil {
		return nil, e
	}
	return tree(s)
}
func Compact(s string) (*model.Node, error) {
	s, e := prepare(s)
	if e != nil {
		return nil, e
	}
	return compact(s)
}

type cursor struct {
	s string
	i int
}

func (p *cursor) err(msg string) error {
	return fmt.Errorf("position %d: %s", utf8.RuneCountInString(p.s[:p.i])+1, msg)
}
func (p *cursor) space() {
	for p.i < len(p.s) {
		r, n := utf8.DecodeRuneInString(p.s[p.i:])
		if !unicode.IsSpace(r) {
			break
		}
		p.i += n
	}
}
func (p *cursor) quoted() (string, error) {
	start := p.i
	p.i++
	for p.i < len(p.s) {
		c := p.s[p.i]
		p.i++
		if c == '\\' {
			if p.i < len(p.s) {
				p.i++
			}
			continue
		}
		if c == '"' {
			var name string
			if e := json.Unmarshal([]byte(p.s[start:p.i]), &name); e != nil {
				return "", p.err("invalid JSON quoted name: " + e.Error())
			}
			if e := model.ValidateName(name); e != nil {
				return "", p.err(e.Error())
			}
			return name, nil
		}
	}
	return "", p.err("unclosed quote")
}

// path reads one or more path components. Tree mode preserves unquoted spaces.
func (p *cursor) path(treeMode bool) ([]string, bool, error) {
	var parts []string
	for {
		if !treeMode {
			p.space()
		}
		if p.i >= len(p.s) {
			return nil, false, p.err("empty path component")
		}
		var name string
		if p.s[p.i] == '"' {
			var e error
			name, e = p.quoted()
			if e != nil {
				return nil, false, e
			}
			if !treeMode {
				p.space()
			}
		} else {
			start := p.i
			for p.i < len(p.s) {
				c := p.s[p.i]
				if c == '/' || (!treeMode && (c == '{' || c == '}' || c == ',')) {
					break
				}
				if c == '"' {
					return nil, false, p.err("quote must begin a component")
				}
				p.i++
			}
			name = p.s[start:p.i]
			if !treeMode {
				name = strings.Map(func(r rune) rune {
					if unicode.IsSpace(r) {
						return -1
					}
					return r
				}, name)
			}
		}
		if e := model.ValidateName(name); e != nil {
			return nil, false, p.err(e.Error())
		}
		parts = append(parts, name)
		if len(parts) > MaxDepth {
			return nil, false, p.err("path too deep")
		}
		if p.i == len(p.s) {
			return parts, false, nil
		}
		if p.s[p.i] != '/' {
			return parts, false, nil
		}
		p.i++
		if !treeMode {
			p.space()
		}
		if p.i == len(p.s) || (!treeMode && strings.ContainsRune("{},", rune(p.s[p.i]))) {
			return parts, true, nil
		}
	}
}

func compact(s string) (*model.Node, error) {
	p := &cursor{s: s}
	parts, dir, e := p.path(false)
	if e != nil {
		return nil, e
	}
	root := &model.Node{Name: parts[0], Dir: true}
	parent := root
	if len(parts) > 1 {
		parent, e = model.Insert(root, parts[1:], true)
		if e != nil {
			return nil, e
		}
	}
	if p.i < len(s) && s[p.i] == '{' {
		if !dir {
			return nil, p.err("directory children require '/{' ")
		}
		e = p.children(parent, len(parts))
		if e != nil {
			return nil, e
		}
	} else if !dir {
		return nil, p.err("root must be a directory ending in '/'")
	}
	p.space()
	if p.i != len(s) {
		return nil, p.err("unexpected trailing input")
	}
	return root, model.Validate(root)
}
func (p *cursor) children(parent *model.Node, depth int) error {
	if depth > MaxDepth {
		return p.err("nesting too deep")
	}
	p.i++
	p.space()
	if p.i < len(p.s) && p.s[p.i] == '}' {
		p.i++
		return nil
	}
	for {
		parts, dir, e := p.path(false)
		if e != nil {
			return e
		}
		n, e := model.Insert(parent, parts, dir)
		if e != nil {
			return p.err(e.Error())
		}
		if p.i < len(p.s) && p.s[p.i] == '{' {
			if !dir {
				return p.err("directory children require '/{' ")
			}
			if e = p.children(n, depth+len(parts)); e != nil {
				return e
			}
		}
		p.space()
		if p.i == len(p.s) {
			return p.err("missing closing brace")
		}
		switch p.s[p.i] {
		case '}':
			p.i++
			return nil
		case ',':
			p.i++
			p.space()
			if p.i == len(p.s) || p.s[p.i] == '}' {
				return p.err("trailing comma or empty element")
			}
		default:
			return p.err("expected comma or closing brace")
		}
	}
}

type row struct {
	line, depth int
	parts       []string
	dir         bool
}

func tree(s string) (*model.Node, error) {
	var rows []row
	for idx, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		depth := 0
		rest := line
		for {
			if strings.HasPrefix(rest, "│   ") {
				depth++
				rest = rest[len("│   "):]
			} else if strings.HasPrefix(rest, "    ") {
				depth++
				rest = rest[4:]
			} else {
				break
			}
		}
		branch := strings.HasPrefix(rest, "├── ") || strings.HasPrefix(rest, "└── ")
		if branch {
			depth++
			rest = rest[len("├── "):]
		} else if len(rows) > 0 || depth > 0 {
			return nil, fmt.Errorf("line %d: expected tree branch prefix", idx+1)
		}
		if len(rows) == 0 && branch {
			return nil, fmt.Errorf("line %d: root cannot have a branch prefix", idx+1)
		}
		if depth > MaxDepth {
			return nil, fmt.Errorf("line %d: nesting too deep", idx+1)
		}
		p := &cursor{s: rest}
		parts, dir, e := p.path(true)
		if e != nil {
			return nil, fmt.Errorf("line %d: %w", idx+1, e)
		}
		if p.i != len(rest) {
			return nil, fmt.Errorf("line %d: unexpected text after quoted name", idx+1)
		}
		rows = append(rows, row{idx + 1, depth, parts, dir})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("input is empty")
	}
	for i := range rows {
		if i+1 < len(rows) && rows[i+1].depth > rows[i].depth {
			if rows[i+1].depth != rows[i].depth+1 {
				return nil, fmt.Errorf("line %d: invalid depth jump", rows[i+1].line)
			}
			rows[i].dir = true
		}
	}
	rows[0].dir = true
	root := &model.Node{Name: rows[0].parts[0], Dir: true}
	last := root
	var e error
	if len(rows[0].parts) > 1 {
		last, e = model.Insert(root, rows[0].parts[1:], true)
		if e != nil {
			return nil, e
		}
	}
	stack := []*model.Node{last}
	for _, r := range rows[1:] {
		if r.depth == 0 || r.depth > len(stack) {
			return nil, fmt.Errorf("line %d: invalid depth or second root", r.line)
		}
		parent := stack[r.depth-1]
		if !parent.Dir {
			return nil, fmt.Errorf("line %d: file has children", r.line)
		}
		n, e := model.Insert(parent, r.parts, r.dir)
		if e != nil {
			return nil, fmt.Errorf("line %d: %w", r.line, e)
		}
		stack = append(stack[:r.depth], n)
	}
	return root, model.Validate(root)
}
