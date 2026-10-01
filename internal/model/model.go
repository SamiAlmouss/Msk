// Package model holds the platform-independent ordered filesystem tree.
package model

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Node struct {
	Name     string
	Dir      bool
	Children []*Node
}

// ValidateName applies Windows naming rules on every platform so portable
// documents cannot create a different structure on Windows.
func ValidateName(s string) error {
	if s == "" || s == "." || s == ".." {
		return fmt.Errorf("invalid path component %q", s)
	}
	if !utf8.ValidString(s) {
		return fmt.Errorf("invalid UTF-8 name")
	}
	if strings.HasSuffix(s, " ") || strings.HasSuffix(s, ".") {
		return fmt.Errorf("name %q ends with a space or dot", s)
	}
	for _, r := range s {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return fmt.Errorf("forbidden character in name %q", s)
		}
	}
	base := strings.ToUpper(strings.SplitN(s, ".", 2)[0])
	base = strings.TrimRight(base, " ")
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$" {
		return fmt.Errorf("reserved device name %q", s)
	}
	if len([]rune(base)) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && strings.ContainsRune("123456789¹²³", []rune(base)[3]) {
		return fmt.Errorf("reserved device name %q", s)
	}
	return nil
}

func Validate(n *Node) error {
	if n == nil || !n.Dir {
		return fmt.Errorf("root must be a directory")
	}
	var walk func(*Node, int) error
	walk = func(n *Node, depth int) error {
		if depth > 256 {
			return fmt.Errorf("structure exceeds 256 levels")
		}
		if err := ValidateName(n.Name); err != nil {
			return err
		}
		if !n.Dir && len(n.Children) > 0 {
			return fmt.Errorf("file %q has children", n.Name)
		}
		seen := map[string]bool{}
		for _, c := range n.Children {
			if c == nil {
				return fmt.Errorf("nil child")
			}
			k := strings.ToLower(c.Name)
			if seen[k] {
				return fmt.Errorf("duplicate or case-conflicting name %q", c.Name)
			}
			seen[k] = true
			if err := walk(c, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(n, 0)
}

// Insert merges shared directories in first-seen order; files are exclusive.
func Insert(parent *Node, parts []string, dir bool) (*Node, error) {
	for i, name := range parts {
		if err := ValidateName(name); err != nil {
			return nil, err
		}
		wantDir := i < len(parts)-1 || dir
		var found *Node
		for _, c := range parent.Children {
			if strings.EqualFold(c.Name, name) {
				if c.Name != name {
					return nil, fmt.Errorf("case-conflicting names %q and %q", c.Name, name)
				}
				found = c
				break
			}
		}
		if found != nil {
			if !found.Dir || !wantDir {
				return nil, fmt.Errorf("duplicate file or file/directory conflict: %q", name)
			}
		} else {
			found = &Node{Name: name, Dir: wantDir}
			parent.Children = append(parent.Children, found)
		}
		parent = found
	}
	return parent, nil
}
