# msk

Free, MIT-licensed project structure CLI, written in Go.

**Windows comes first:** the ready-made `msk.exe` requires no Go, Python, Rust,
.NET, npm, or Gradle.

`msk` creates **directories and empty files only**, converts between text trees
and a compact brace notation, and scans existing directories without reading
file contents.

Existing files keep their contents and permissions. Nothing is deleted or
overwritten inside a generated project.

---

## 🚀 Installation

### Windows PowerShell

Install the latest version of `msk` with one command:

```powershell
irm https://msk-install.pages.dev | iex
```

The installer:

- Downloads the latest `msk.exe` release from GitHub.
- Installs it into `%LOCALAPPDATA%\msk\bin`.
- Adds the installation directory to the current user's `PATH`.
- Does not require Administrator privileges.
- Can be run again to update or reinstall `msk`.

After installation, verify it with:

```powershell
msk --help
```

or:

```powershell
msk --version
```

### Installation location

On Windows, `msk` is installed to:

```text
%LOCALAPPDATA%\msk\bin\msk.exe
```

### Direct installer source

The short installation URL redirects to the installer stored in this repository:

```text
https://raw.githubusercontent.com/SamiAlmouss/Msk/main/scripts/install.ps1
```

You can also run the installer directly:

```powershell
irm https://raw.githubusercontent.com/SamiAlmouss/Msk/main/scripts/install.ps1 | iex
```

> [!NOTE]
> Running a remote PowerShell script executes code from the internet.
> You can review `scripts/install.ps1` in this repository before running it.

### Direct executable download

The latest Windows executable is available from GitHub Releases:

```text
https://github.com/SamiAlmouss/Msk/releases/latest/download/msk.exe
```

GitHub Releases:

```text
https://github.com/SamiAlmouss/Msk/releases
```

---

## ⚡ Quick start

Run commands from the directory where the new project belongs:

```powershell
msk --help
msk --version
msk                         # Read Windows Clipboard, create in current directory
msk tree.txt                # Any UTF-8 source filename or path is accepted
msk --preview               # Clipboard, validation and full plan; no writes
msk tree.txt --preview
msk tree.txt --output "D:\Projects"
```

For a root named `TVSnake/`, running in:

```text
D:\Projects
```

creates:

```text
D:\Projects\TVSnake
```

The working directory never changes and the executable's directory is irrelevant.

`--output` may be absolute; missing destination directories are planned and
created.

Success reports:

- Created directories
- Created files
- Existing items kept
- Destination

Failure during execution lists paths created before the failure; execution is
not atomic.

---

## 📋 Create from Clipboard

Copy a project structure:

```text
TestProject/
├── README.md
├── src/
│   ├── main.go
│   └── utils.go
└── docs/
    └── usage.md
```

Open PowerShell in the directory where you want the project to be created and run:

```powershell
msk
```

`msk` reads the structure directly from the Windows Clipboard and creates it in
the current directory.

---

## 📄 Create from a file

Save a tree structure to any UTF-8 text file, for example:

```text
tree.txt
```

Then run:

```powershell
msk tree.txt
```

You can also specify another output directory:

```powershell
msk tree.txt --output "D:\Projects"
```

Preview the result without creating anything:

```powershell
msk tree.txt --preview
```

---

## 🔄 Convert and scan

`msk` can convert between tree and compact formats and can generate a structure
description from an existing directory:

```powershell
msk convert tree.txt --compact
msk convert compact.txt --tree
msk convert --clipboard --compact
msk convert --clipboard --tree
msk convert tree.txt --compact --save compact.txt
msk convert tree.txt --compact --save compact.txt --overwrite-output

msk scan . --tree
msk scan . --compact
msk scan "D:\Projects\TVSnake" --tree
msk scan . --tree --save tree.txt
msk scan . --compact --copy
msk scan . --tree --exclude .git --exclude node_modules
msk scan . --tree --max-depth 3
```

`scan` defaults to tree format.

`convert` requires exactly one of:

```text
--tree
--compact
```

Both commands also accept:

```text
--copy
--save FILE
--overwrite-output
```

Options work before and after positional arguments.

`--` ends option parsing. Use it for filenames beginning with `-` or named
`scan` / `convert`, for example:

```powershell
msk --preview -- -tree.txt
```

Unknown options, conflicting formats, and extra arguments fail.

No interactive prompts are required.

Converted/scanned text goes to stdout with no banners or colors.

Compact output contains no newline; tree output ends with a newline.

Diagnostics, save/copy confirmations, and depth warnings go to stderr.

`--save` writes UTF-8 without BOM; stdout still contains the result.

Output files are exclusive by default; the explicit overwrite switch affects
only conversion/scan output.

Results are staged, flushed, then published. Exclusive publication uses a hard
link and requires a filesystem supporting hard links (NTFS on Windows);
overwrite uses a rename.

PowerShell 5.1 redirection may re-encode native output; prefer `--save` for
UTF-8.

### Exit codes

| Code | Meaning |
|---:|---|
| `0` | Success |
| `1` | Filesystem, Clipboard, or execution failure |
| `2` | Command usage or invalid structure text |

---

# Tree format

A tree document contains one directory root, with `├── ` or `└── ` before
descendants.

Each indentation unit is exactly:

```text
│   
```

or four spaces.

Indentation is counted as text columns rather than UTF-8 bytes.

LF, CRLF, UTF-8 BOM, and blank lines are accepted.

Invalid indentation jumps and unexpected extra text are errors and include line
numbers.

Example:

```text
TVSnake/
├── settings.gradle.kts
├── build.gradle.kts
├── gradle.properties
└── app/
    ├── build.gradle.kts
    └── src/main/
        ├── AndroidManifest.xml
        ├── java/com/example/tvsvnake/
        │   ├── MainActivity.kt
        │   └── GameView.kt
        └── res/
            ├── values/strings.xml
            └── drawable/app_banner.xml
```

Paths in documents use `/`, even on Windows.

A trailing `/` marks a directory.

An item with children is also a directory.

Other leaf items are files regardless of extension, so names such as these work:

```text
Dockerfile
LICENSE
README
```

An empty directory needs its trailing `/`.

The root is always a directory.

Intermediate components expand into directories.

Shared directories such as:

```text
src/a.go
src/b.go
```

merge in first-seen order.

Repeated file definitions and file/directory conflicts are rejected.

Names keep internal and leading spaces.

Trailing spaces are rejected by Windows naming rules.

No suffix comments are interpreted: `#` belongs to the name.

A complete outer Markdown fence of exactly three backticks, optionally followed
by `text`, may wrap the input.

Extra text outside the fence is rejected.

---

## Quoted tree components

JSON double-quoted **individual components** can represent names with tree
characters or leading spaces without ambiguity:

```text
"My Project"/
├── " source files"/
│   └── main.go
├── "notes, draft.txt"
└── "├── name"/
```

Output expands every level separately, uses four columns per indentation unit,
marks every directory with `/`, and quotes names when needed.

Empty directories, types, names, and sibling order survive conversion.

---

# Compact format

The same structure can be represented using compact brace notation:

```text
TVSnake/{settings.gradle.kts,build.gradle.kts,gradle.properties,app/{build.gradle.kts,src/main/{AndroidManifest.xml,java/com/example/tvsvnake/{MainActivity.kt,GameView.kt},res/{values/strings.xml,drawable/app_banner.xml}}}}
```

Directories use:

```text
name/{children}
```

Commas separate children and `/` separates path components.

An empty directory can be written as:

```text
empty/
```

An empty root can be:

```text
EmptyProject/
```

This is also accepted:

```text
name/{}
```

Whitespace outside quoted names is formatting and is ignored.

Use quotes to preserve spaces:

```text
"My Project"/{README.md,"source files"/{main.go},"notes, draft.txt",empty/}
```

Quotes follow JSON escape rules.

For example:

```text
"\u0645\u0644\u0641"
```

represents an Arabic name.

A quoted component is one name, never a whole path.

Encoded or literal `/` and `\` are prohibited inside names.

Quotes do not bypass Windows validation. For example, a decoded double quote
remains forbidden.

Missing braces, unclosed quotes, bad escapes, empty elements, extra commas, and
invalid paths report a position.

Nesting and document size are bounded at:

```text
256 levels
16 MiB
```

---

# Validation and filesystem safety

Windows naming rules are applied on all platforms for portable documents.

The following are rejected:

- Absolute paths
- UNC paths
- Drive letters
- `.` and `..`
- Empty components
- Control/NUL characters
- `< > : " / \ | ? *`
- Reserved Windows device names, including extensions
- Trailing spaces or dots
- Case-conflicting siblings

Names are never sanitized or silently changed.

Names accepted by the text model may still exceed filesystem length limits,
which results in a clear I/O error.

All input is validated before disk access for creation.

A complete plan checks existing types and ancestor paths before writing.

Existing files are kept.

New files use exclusive creation.

Every existing ancestor is inspected with `Lstat`, and Windows
`FILE_ATTRIBUTE_REPARSE_POINT` covers junctions as well as symlinks.

Checks run:

1. During planning
2. Again before execution
3. Before each item

No existing permissions are changed.

Scan skips links/reparse points with stderr warnings, never follows them, and
refuses a linked root or ancestor.

These path checks narrow races but cannot eliminate concurrent malicious
replacement between checking a path and opening/creating it.

Do not generate or save into directories concurrently modified by untrusted
processes.

This version does not use kernel handle-relative traversal and does not claim
full race protection.

---

## Scan behavior

Scan sorts directories first, then files, using case-sensitive Go string order
(lexicographic UTF-8 bytes) within each group.

Hidden names are included.

Repeated:

```text
--exclude NAME
```

matches the exact case-sensitive basename at any level, with no glob expansion.

Root depth is zero.

Directories at `--max-depth` appear empty and stderr states that the result is
depth-limited.

Recreating a depth-limited result does not recreate omitted descendants.

Access failures and unrepresentable names fail the scan rather than silently
producing a partial successful result.

When `--save` is inside the scanned directory, its exact output path is excluded
automatically, including when the output already exists.

---

# Clipboard

Clipboard access is isolated behind an interface.

On Windows it uses:

- Native Windows Unicode APIs
- A hidden owner window
- Retry logic when Clipboard access is busy
- No shell evaluation of Clipboard text

Clipboard errors and an empty/non-text Clipboard fail clearly.

Other operating systems support:

- File input
- Scan
- Conversion

Clipboard input is currently Windows-only.

---

# 🔄 Update

To update or reinstall `msk`, simply run the installer again:

```powershell
irm https://msk-install.pages.dev | iex
```

The installer downloads the executable from the latest GitHub Release and
replaces the installed copy.

You do not need to manually remove the previous version first.

---

# 🗑️ Uninstall

The repository includes an uninstall script:

```powershell
.\scripts\uninstall.ps1
```

It removes only:

- `msk.exe`
- Empty `msk` installation directories
- The `msk` bin directory from the user `PATH`

Other files and unrelated `PATH` entries are kept.

It does not permanently change PowerShell ExecutionPolicy.

If you cloned the repository, run:

```powershell
& .\scripts\uninstall.ps1
```

The default installation directory is:

```text
%LOCALAPPDATA%\msk\bin
```

---

# 📦 Releases

Public releases are published through GitHub Releases:

```text
https://github.com/SamiAlmouss/Msk/releases
```

The latest executable can be downloaded directly from:

```text
https://github.com/SamiAlmouss/Msk/releases/latest/download/msk.exe
```

Current initial public release:

```text
v0.1.0
```

The short installer endpoint is:

```text
https://msk-install.pages.dev
```

which allows installation with:

```powershell
irm https://msk-install.pages.dev | iex
```

---

# Build and test

The project uses **Go 1.26.8**, with language baseline `go 1.26.0` in `go.mod`
and the exact compiler version pinned in CI.

No external modules are used, so no `go.sum` is needed.

```powershell
go test ./...
go vet ./...
go build -o msk.exe ./cmd/msk

.\msk.exe --help
.\msk.exe examples\tree.txt --preview
.\msk.exe examples\tree.txt --output .\sandbox
.\msk.exe convert examples\compact.txt --tree
.\msk.exe scan .\sandbox\TVSnake --compact

go test ./internal/parser -run '^$' -fuzz FuzzParsers -fuzztime 10s -parallel 2

& .\scripts\build.ps1 -Version v1.0.0 -Commit unknown
& .\scripts\test-install.ps1 -Version v1.0.0
```

Build output includes:

```text
dist/windows_amd64/msk.exe
dist/windows_arm64/msk.exe
```

`build.ps1` restores environment settings after building and embeds:

- Version
- Commit
- UTC build date

The release build system can create:

```text
msk_v1.0.0_windows_amd64.zip
msk_v1.0.0_windows_arm64.zip
checksums.txt
```

`test-install.ps1` uses local release assets and an isolated temporary
`LOCALAPPDATA`, disables real PATH writes, and tests:

- Install
- Reinstall
- Checksum failure
- Preservation of the existing binary
- PATH string operations
- Uninstall

Run it in both PowerShell editions; CI does so.

It does not require Pester.

Tests cover:

- Examples
- Unicode
- Quoting
- Short paths
- Directory merging
- Ordering
- Invalid input
- Round trips
- Parser fuzzing
- CLI streams and exit codes
- Temporary-disk creation
- Existing file preservation
- No-write preview/preflight
- Scan filters
- Scan depth
- Save exclusion
- Links

Symlink/junction tests explicitly skip if platform privileges prevent setup.

Clipboard unit tests use a fake and do not disturb the user's real Clipboard.

A Windows native Clipboard test allocates a private window station and desktop,
so it never reads or modifies the user's Clipboard.

It skips explicitly if Windows denies creation of that isolated desktop.

Interactive desktop Clipboard and native ARM64 execution require their
respective environments; cross compilation alone does not establish native
runtime behavior.

---

# GitHub Actions and releases

The project includes:

```text
.github/workflows/ci.yml
.github/workflows/release.yml
```

CI is used to test the project automatically.

The release workflow is designed to build release artifacts when a version tag
is published.

Typical release flow:

```powershell
git add .
git commit -m "Prepare release"
git push
```

Create a version tag:

```powershell
git tag v0.1.1
```

Push the tag:

```powershell
git push origin v0.1.1
```

The release workflow can then build and publish the appropriate release
artifacts.

The installation URL does not need to change between versions:

```powershell
irm https://msk-install.pages.dev | iex
```

because the installer uses the latest published GitHub release.

CI uses read-only repository permissions.

Only the tag release job requires:

```text
contents: write
```

to publish releases.

No additional repository secrets are required for normal GitHub release
publishing with `GITHUB_TOKEN`.

---

# Source layout

```text
Msk/
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── cmd/
│   └── msk/
│       └── main.go
├── examples/
│   ├── compact.txt
│   └── tree.txt
├── internal/
│   ├── cli/
│   ├── clipboard/
│   ├── filesystem/
│   ├── formatter/
│   ├── model/
│   └── parser/
├── scripts/
│   ├── build.ps1
│   ├── install.ps1
│   ├── test-install.ps1
│   └── uninstall.ps1
├── .gitignore
├── LICENSE
├── README.md
└── go.mod
```

`cmd/msk` is the executable entry point.

`internal/cli` handles options and sources.

`internal/model` validates the ordered tree.

`internal/parser` parses both grammars.

`internal/formatter` serializes them.

`internal/filesystem` plans, executes, scans, and stages output.

`internal/clipboard` contains the Windows-specific Clipboard adapter.

`examples` contains sample tree and compact documents.

`scripts` contains build, install, uninstall, and distribution test scripts.

`.github` contains CI and release workflows.

Platform-independent logic uses only the Go standard library.

Shared directory declarations merge; duplicate files never do.

---

# Verification of this implementation

The implementation was tested locally on Windows amd64 with Go 1.26.8.

### Tests performed

- `go test -count=1 ./...` passed all available tests.
- Native symlink creation skipped where the current Windows token lacked that
  privilege.
- Junction safety tests passed.
- `go vet ./...` passed.
- Parser fuzzing with round-trip assertions ran for 10 seconds with two workers.
- Release builds for Windows amd64 and arm64 were created locally.
- Build metadata and SHA-256 generation were tested.
- The actual amd64 executable was tested with:
  - Help
  - Version
  - Creation from both TVSnake examples
  - Scan comparison
  - Repeat execution
- Both text formats produced the same structure.
- Repeat execution preserved existing project items.
- Offline installer tests passed in PowerShell 7 and Windows PowerShell 5.1.
- Install/reinstall behavior was tested.
- Locked-file rejection was tested.
- Missing/corrupt checksum rejection was tested by the distribution test suite.
- Existing binary preservation was tested.
- PATH deduplication/removal was tested.
- Uninstall behavior was tested.
- A public GitHub release has been created.
- Installation through the published GitHub-hosted installer has been tested.
- The short Cloudflare Pages installation endpoint is available at
  `https://msk-install.pages.dev`.

Native ARM64 runtime behavior still requires testing on ARM64 hardware.

---

# Requirements

For the prebuilt Windows release:

- Windows 10 or Windows 11
- PowerShell
- Internet connection during installation

No development runtime is required to use the prebuilt executable.

You do **not** need:

- Go
- Python
- Rust
- .NET SDK
- Node.js
- npm
- Gradle

Go is required only if you want to build `msk` from source.

---

# License

Licensed under the [MIT License](LICENSE).

---

# Author

Created by **SamiAlmouss**

GitHub:

```text
https://github.com/SamiAlmouss
```

Project repository:

```text
https://github.com/SamiAlmouss/Msk
```

---

## ⭐ Support

If you find `msk` useful, consider giving the repository a ⭐ on GitHub.
