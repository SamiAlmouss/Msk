// Package filesystem plans creation separately from execution and scans names only.
package filesystem

import (
	"fmt"
	"io"
	"msk/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Item struct {
	Path     string
	Dir      bool
	Existing bool
}
type Plan struct {
	Destination string
	Items       []Item
}
type Result struct {
	Directories, Files, Skipped int
	Created                     []string
}

// CheckPath checks every existing ancestor, including the requested path.
// Rechecking narrows, but cannot eliminate, concurrent path replacement races.
func CheckPath(path string) error {
	a, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	var paths []string
	for {
		paths = append(paths, a)
		p := filepath.Dir(a)
		if p == a {
			break
		}
		a = p
	}
	for i := len(paths) - 1; i >= 0; i-- {
		info, e := os.Lstat(paths[i])
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return fmt.Errorf("inspect %s: %w", paths[i], e)
		}
		link, e := isLink(paths[i], info)
		if e != nil {
			return e
		}
		if link {
			return fmt.Errorf("refusing link or reparse point: %s", paths[i])
		}
		if i > 0 && !info.IsDir() {
			return fmt.Errorf("ancestor is not a directory: %s", paths[i])
		}
	}
	return nil
}
func inspect(path string, dir bool) (bool, error) {
	if e := CheckPath(path); e != nil {
		return false, e
	}
	info, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if info.IsDir() != dir {
		return false, fmt.Errorf("file/directory conflict: %s", path)
	}
	if !dir && !info.Mode().IsRegular() {
		return false, fmt.Errorf("not a regular file: %s", path)
	}
	return true, nil
}
func Prepare(n *model.Node, output string) (*Plan, error) {
	if e := model.Validate(n); e != nil {
		return nil, e
	}
	base, e := filepath.Abs(output)
	if e != nil {
		return nil, e
	}
	if e = CheckPath(base); e != nil {
		return nil, e
	}
	p := &Plan{Destination: filepath.Join(base, n.Name)}
	// Include missing destination ancestors in the plan, without changing existing permissions.
	var missing []string
	for a := base; ; a = filepath.Dir(a) {
		info, e := os.Lstat(a)
		if e == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("output is not a directory: %s", a)
			}
			break
		}
		if !os.IsNotExist(e) {
			return nil, e
		}
		missing = append(missing, a)
		if filepath.Dir(a) == a {
			return nil, fmt.Errorf("missing filesystem root")
		}
	}
	for i := len(missing) - 1; i >= 0; i-- {
		p.Items = append(p.Items, Item{Path: missing[i], Dir: true})
	}
	var walk func(*model.Node, string) error
	walk = func(n *model.Node, parent string) error {
		path := filepath.Join(parent, n.Name)
		rel, e := filepath.Rel(base, path)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("path escapes destination")
		}
		exists, e := inspect(path, n.Dir)
		if e != nil {
			return e
		}
		p.Items = append(p.Items, Item{path, n.Dir, exists})
		for _, c := range n.Children {
			if e = walk(c, path); e != nil {
				return e
			}
		}
		return nil
	}
	if e = walk(n, base); e != nil {
		return nil, e
	}
	return p, nil
}
func (p *Plan) Preview(w io.Writer) {
	for _, it := range p.Items {
		action := "Create file"
		if it.Dir {
			action = "Create directory"
		}
		if it.Existing {
			action = "Keep existing"
		}
		fmt.Fprintf(w, "%s: %s\n", action, it.Path)
	}
	fmt.Fprintln(w, "Destination:", p.Destination)
}
func (p *Plan) Execute() (r Result, err error) {
	// A complete second preflight catches changes since Prepare before first write.
	for _, it := range p.Items {
		if _, e := inspect(it.Path, it.Dir); e != nil {
			return r, e
		}
	}
	for _, it := range p.Items {
		exists, e := inspect(it.Path, it.Dir)
		if e != nil {
			return r, e
		}
		if exists {
			r.Skipped++
			continue
		}
		if it.Dir {
			e = os.Mkdir(it.Path, 0755)
		} else {
			var f *os.File
			f, e = os.OpenFile(it.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
			if e == nil {
				r.Files++
				r.Created = append(r.Created, it.Path)
				e = f.Close()
				if e != nil {
					return r, e
				}
				continue
			}
		}
		if e != nil {
			if os.IsExist(e) {
				ok, check := inspect(it.Path, it.Dir)
				if check == nil && ok {
					r.Skipped++
					continue
				}
			}
			return r, fmt.Errorf("create %s: %w", it.Path, e)
		}
		r.Directories++
		r.Created = append(r.Created, it.Path)
	}
	return r, nil
}

type ScanOptions struct {
	Exclude  []string
	MaxDepth int
	SavePath string
	Warnings io.Writer
}

func Scan(path string, opt ScanOptions) (*model.Node, error) {
	base, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	if e = CheckPath(base); e != nil {
		return nil, e
	}
	info, e := os.Lstat(base)
	if e != nil {
		return nil, e
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan root is not a directory: %s", base)
	}
	save := ""
	if opt.SavePath != "" {
		save, e = filepath.Abs(opt.SavePath)
		if e != nil {
			return nil, e
		}
	}
	var walk func(string, string, int) (*model.Node, error)
	walk = func(path, name string, depth int) (*model.Node, error) {
		if depth > 256 {
			return nil, fmt.Errorf("cannot represent %s: structure exceeds 256 levels", path)
		}
		if e := model.ValidateName(name); e != nil {
			return nil, fmt.Errorf("cannot represent %s: %w", path, e)
		}
		if e := CheckPath(path); e != nil {
			return nil, e
		}
		n := &model.Node{Name: name, Dir: true}
		if opt.MaxDepth >= 0 && depth >= opt.MaxDepth {
			return n, nil
		}
		entries, e := os.ReadDir(path)
		if e != nil {
			return nil, fmt.Errorf("read %s: %w", path, e)
		}
		var children []*model.Node
		for _, ent := range entries {
			excluded := false
			for _, x := range opt.Exclude {
				if ent.Name() == x {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
			cp := filepath.Join(path, ent.Name())
			if save != "" && samePath(cp, save) {
				continue
			}
			ci, e := os.Lstat(cp)
			if e != nil {
				return nil, fmt.Errorf("inspect %s: %w", cp, e)
			}
			link, e := isLink(cp, ci)
			if e != nil {
				return nil, e
			}
			if link {
				if opt.Warnings != nil {
					fmt.Fprintln(opt.Warnings, "Skipping link:", cp)
				}
				continue
			}
			var c *model.Node
			if ci.IsDir() {
				c, e = walk(cp, ent.Name(), depth+1)
			} else {
				if !ci.Mode().IsRegular() {
					return nil, fmt.Errorf("cannot represent special file: %s", cp)
				}
				e = model.ValidateName(ent.Name())
				c = &model.Node{Name: ent.Name()}
			}
			if e != nil {
				return nil, fmt.Errorf("scan %s: %w", cp, e)
			}
			children = append(children, c)
		}
		sort.Slice(children, func(i, j int) bool {
			if children[i].Dir != children[j].Dir {
				return children[i].Dir
			}
			return children[i].Name < children[j].Name
		})
		n.Children = children
		return n, nil
	}
	n, e := walk(base, filepath.Base(base), 0)
	if e != nil {
		return nil, e
	}
	return n, model.Validate(n)
}

// Save stages bytes beside the destination. Exclusive hard-link publication
// prevents a concurrent existing output from being overwritten by default.
func Save(path, text string, overwrite bool) error {
	a, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if e = CheckPath(a); e != nil {
		return e
	}
	if info, e := os.Lstat(a); e == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("output is not a regular file: %s", a)
		}
		if !overwrite {
			return fmt.Errorf("output exists; use --overwrite-output: %s", a)
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(a), ".msk-output-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, e = f.WriteString(text); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = CheckPath(a); e != nil {
		return e
	}
	if overwrite {
		return os.Rename(tmp, a)
	}
	if e = os.Link(tmp, a); e != nil {
		return fmt.Errorf("publish output (requires hard-link capable filesystem): %w", e)
	}
	return nil
}
