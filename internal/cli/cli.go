// Package cli provides a testable command line entry point.
package cli

import (
	"fmt"
	"io"
	"msk/internal/clipboard"
	"msk/internal/filesystem"
	"msk/internal/formatter"
	"msk/internal/model"
	"msk/internal/parser"
	"os"
	"strconv"
	"strings"
)

type App struct {
	Clipboard             clipboard.Interface
	Out, Err              io.Writer
	Version, Commit, Date string
}
type options struct {
	command                                                      string
	args                                                         []string
	tree, compact, preview, clip, copy, overwrite, help, version bool
	output, save                                                 string
	exclude                                                      []string
	depth                                                        int
	seen                                                         map[string]bool
}

func parseArgs(args []string) (o options, e error) {
	o.command = "create"
	o.output = "."
	o.depth = -1
	o.seen = map[string]bool{}
	end := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !end && arg == "--" {
			end = true
			continue
		}
		if !end && strings.HasPrefix(arg, "-") {
			key, val, has := strings.Cut(arg, "=")
			o.seen[key] = true
			value := func() (string, error) {
				if has {
					return val, nil
				}
				if i+1 >= len(args) {
					return "", fmt.Errorf("%s requires a value", key)
				}
				i++
				return args[i], nil
			}
			boolFlag := true
			switch key {
			case "--help", "-h":
				o.help = true
			case "--version":
				o.version = true
			case "--tree":
				o.tree = true
			case "--compact":
				o.compact = true
			case "--preview":
				o.preview = true
			case "--clipboard":
				o.clip = true
			case "--copy":
				o.copy = true
			case "--overwrite-output":
				o.overwrite = true
			case "--output":
				boolFlag = false
				o.output, e = value()
			case "--save":
				boolFlag = false
				o.save, e = value()
			case "--exclude":
				boolFlag = false
				var v string
				v, e = value()
				o.exclude = append(o.exclude, v)
			case "--max-depth":
				boolFlag = false
				var v string
				v, e = value()
				if e == nil {
					o.depth, e = strconv.Atoi(v)
					if e != nil || o.depth < 0 {
						e = fmt.Errorf("--max-depth requires a nonnegative integer")
					}
				}
			default:
				return o, fmt.Errorf("unknown option %s", key)
			}
			if e != nil {
				return o, e
			}
			if boolFlag && has {
				return o, fmt.Errorf("%s does not accept a value", key)
			}
			continue
		}
		if !end && len(o.args) == 0 && o.command == "create" && (arg == "scan" || arg == "convert") {
			o.command = arg
			continue
		}
		o.args = append(o.args, arg)
	}
	if o.tree && o.compact {
		return o, fmt.Errorf("--tree and --compact are mutually exclusive")
	}
	if o.help || o.version {
		if o.help && o.version {
			return o, fmt.Errorf("--help and --version are mutually exclusive")
		}
		return o, nil
	}
	if len(o.args) > 1 {
		return o, fmt.Errorf("too many positional arguments")
	}
	if o.clip && len(o.args) > 0 {
		return o, fmt.Errorf("--clipboard conflicts with a source file")
	}
	if o.overwrite && o.save == "" {
		return o, fmt.Errorf("--overwrite-output requires --save")
	}
	allowed := map[string]bool{"--help": true, "-h": true, "--version": true}
	var keys []string
	switch o.command {
	case "create":
		keys = []string{"--output", "--preview"}
	case "convert":
		keys = []string{"--tree", "--compact", "--save", "--copy", "--overwrite-output", "--clipboard"}
		if !o.tree && !o.compact {
			return o, fmt.Errorf("convert requires --tree or --compact")
		}
		if len(o.args) == 0 && !o.clip {
			return o, fmt.Errorf("convert requires a source file or --clipboard")
		}
	case "scan":
		keys = []string{"--tree", "--compact", "--save", "--copy", "--overwrite-output", "--exclude", "--max-depth"}
		if len(o.args) != 1 {
			return o, fmt.Errorf("scan requires a directory")
		}
	}
	for _, k := range keys {
		allowed[k] = true
	}
	for k := range o.seen {
		if !allowed[k] {
			return o, fmt.Errorf("%s is not valid for %s", k, o.command)
		}
	}
	if o.output == "" || o.seen["--save"] && o.save == "" {
		return o, fmt.Errorf("output path cannot be empty")
	}
	return o, nil
}

func (a App) Run(args []string) int {
	if a.Out == nil {
		a.Out = io.Discard
	}
	if a.Err == nil {
		a.Err = io.Discard
	}
	if a.Clipboard == nil {
		a.Clipboard = clipboard.System{}
	}
	fail := func(code int, e error) int { fmt.Fprintln(a.Err, "msk:", e); return code }
	o, e := parseArgs(args)
	if e != nil {
		return fail(2, e)
	}
	if o.help {
		fmt.Fprint(a.Out, Help)
		return 0
	}
	if o.version {
		fmt.Fprintf(a.Out, "msk %s (commit %s, built %s)\n", a.Version, a.Commit, a.Date)
		return 0
	}
	var n *model.Node
	if o.command == "scan" {
		n, e = filesystem.Scan(o.args[0], filesystem.ScanOptions{Exclude: o.exclude, MaxDepth: o.depth, SavePath: o.save, Warnings: a.Err})
		if e != nil {
			return fail(1, e)
		}
		if o.depth >= 0 {
			fmt.Fprintf(a.Err, "Scan limited to depth %d (root is depth 0).\n", o.depth)
		}
	} else {
		var source string
		if len(o.args) == 0 {
			source, e = a.Clipboard.Read()
		} else {
			source, e = readSource(o.args[0])
		}
		if e != nil {
			return fail(1, e)
		}
		n, e = parser.Parse(source)
		if e != nil {
			return fail(2, e)
		}
	}
	if o.command == "create" {
		plan, e := filesystem.Prepare(n, o.output)
		if e != nil {
			return fail(1, e)
		}
		if o.preview {
			plan.Preview(a.Out)
			return 0
		}
		r, e := plan.Execute()
		if e != nil {
			for _, path := range r.Created {
				fmt.Fprintln(a.Err, "Created before failure:", path)
			}
			return fail(1, e)
		}
		fmt.Fprintf(a.Out, "Created directories: %d\nCreated files: %d\nSkipped existing items: %d\nDestination: %s\n", r.Directories, r.Files, r.Skipped, plan.Destination)
		return 0
	}
	text := formatter.Tree(n)
	if o.compact {
		text = formatter.Compact(n)
	}
	if o.save != "" {
		if e = filesystem.Save(o.save, text, o.overwrite); e != nil {
			return fail(1, e)
		}
		fmt.Fprintln(a.Err, "Saved:", o.save)
	}
	if o.copy {
		if e = a.Clipboard.Write(text); e != nil {
			return fail(1, e)
		}
		fmt.Fprintln(a.Err, "Copied to Clipboard.")
	}
	if _, e = io.WriteString(a.Out, text); e != nil {
		return fail(1, e)
	}
	return 0
}
func readSource(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, parser.MaxInput+1))
	if e != nil {
		return "", e
	}
	return string(b), nil
}

const Help = `msk - create, convert, and scan project structures

Usage:
  msk [source-file] [--preview] [--output DIRECTORY]
  msk convert FILE --tree|--compact [--save FILE] [--copy]
  msk convert --clipboard --tree|--compact [--save FILE] [--copy]
  msk scan DIRECTORY [--tree|--compact] [--exclude NAME] [--max-depth N]
           [--save FILE] [--copy] [--overwrite-output]
  msk --help
  msk --version

Without a source file, creation reads Windows Clipboard. New files are empty;
existing files are preserved. Preview validates and plans without writing.
Tree format uses one directory root and four-column indentation:
  Demo/
  ├── README.md
  └── empty/
Compact format: Demo/{README.md,empty/}
Use JSON quoted path components for spaces, commas, braces, or tree symbols:
  "My Project"/{"notes, draft.txt",empty/}
Text paths use /; quoted names cannot contain path separators.

Examples (PowerShell):
  msk tree.txt --preview
  msk tree.txt --output "D:\Projects"
  msk convert tree.txt --compact --save compact.txt
  msk scan . --tree --exclude .git --exclude node_modules --max-depth 3
  msk scan "D:\Projects\TVSnake" --compact --copy

Options may precede or follow positional arguments. Use -- before filenames
beginning with a dash or named scan/convert. --exclude matches exact names at
every level (no glob). scan defaults to tree; root depth is zero.
--overwrite-output only replaces conversion/scan output, never project files.
Exit codes: 0 success; 1 I/O, Clipboard or execution failure; 2 usage or invalid text.
`
