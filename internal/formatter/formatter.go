package formatter

import (
	"encoding/json"
	"msk/internal/model"
	"strings"
	"unicode"
)

func name(s string) string {
	quote := false
	for _, r := range s {
		if unicode.IsSpace(r) || strings.ContainsRune(`{},"\\├└│─`, r) {
			quote = true
		}
	}
	if quote {
		b, _ := json.Marshal(s)
		return string(b)
	}
	return s
}
func Compact(n *model.Node) string {
	s := name(n.Name)
	if !n.Dir {
		return s
	}
	s += "/"
	if len(n.Children) == 0 {
		return s
	}
	parts := make([]string, len(n.Children))
	for i, c := range n.Children {
		parts[i] = Compact(c)
	}
	return s + "{" + strings.Join(parts, ",") + "}"
}
func Tree(n *model.Node) string {
	var b strings.Builder
	var walk func(*model.Node, string, string)
	walk = func(n *model.Node, prefix, branch string) {
		b.WriteString(prefix + branch + name(n.Name))
		if n.Dir {
			b.WriteByte('/')
		}
		b.WriteByte('\n')
		for i, c := range n.Children {
			br := "├── "
			if i == len(n.Children)-1 {
				br = "└── "
			}
			p := prefix
			if branch != "" {
				if branch == "└── " {
					p += "    "
				} else {
					p += "│   "
				}
			}
			walk(c, p, br)
		}
	}
	walk(n, "", "")
	return b.String()
}
