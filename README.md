# msk

Free, MIT-licensed project structure CLI, written in Go. Windows comes first:
the ready-made `msk.exe` needs no Go, Python, Rust, .NET, npm, or Gradle.
It creates **directories and empty files only**, converts between text trees
and a brace notation, and scans existing directories without reading file contents.
Existing files keep their contents and permissions. Nothing is deleted or overwritten
inside a generated project. Source, tests, scripts and CI are included; no release
or public download has been published as part of creating this project.

## Quick start

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

For a root named `TVSnake/`, running in `D:\Projects` creates `D:\Projects\TVSnake`.
The working directory never changes and the executable's directory is irrelevant.
`--output` may be absolute; missing destination directories are planned and created.
Success reports created directories, created files, existing items kept, and destination.
Failure during execution lists paths created before the failure; execution is not atomic.

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

`scan` defaults to tree; `convert` requires exactly one of `--tree` and `--compact`.
Both commands also accept `--copy`, `--save FILE`, and `--overwrite-output`.
Options work before and after positional arguments. `--` ends options; use it
for filenames beginning with `-` or named `scan`/`convert`, e.g.
`msk --preview -- -tree.txt`. Unknown options, conflicting formats and extra
arguments fail. No interactive prompts are required.

Converted/scanned text goes to stdout with no banners or colors. Compact output
contains no newline; tree output ends with a newline. Diagnostics, save/copy
confirmations and depth warnings go to stderr. `--save` writes UTF-8 without BOM;
stdout still contains the result. Output files are exclusive by default; the
explicit overwrite switch affects only conversion/scan output. Results are staged,
flushed, then published. Exclusive publication uses a hard link and requires a
filesystem supporting hard links (NTFS on Windows); overwrite uses a rename.
PowerShell 5.1 redirection may re-encode native output; prefer `--save` for UTF-8.

Exit codes: **0** success; **1** filesystem, Clipboard or execution failure;
**2** command usage or invalid structure text.

## Tree format

One directory root, with `├── ` or `└── ` before descendants. Each indentation
unit is exactly `│   ` or four spaces, counted as text columns rather than UTF-8
bytes. LF, CRLF, UTF-8 BOM and blank lines are accepted. Invalid indentation
jumps and unexpected extra text are errors, with line numbers.

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

Paths in documents use `/`, even on Windows. A trailing `/` marks a directory;
an item with children is also a directory. Other leaf items are files, regardless
of extension: `Dockerfile` and `LICENSE` work. An empty directory needs its `/`.
The root is always a directory. Intermediate components expand into directories.
Shared directories such as `src/a.go` and `src/b.go` merge in first-seen order.
Repeated file definitions and file/directory conflicts are rejected.

Names keep internal and leading spaces; trailing spaces are rejected by Windows
naming rules. No suffix comments are interpreted: `#` belongs to the name.
A complete outer Markdown fence of exactly three backticks, optionally followed
by `text`, may wrap the input; extra text outside the fence is rejected.

Tree extension: JSON double-quoted **individual components** can represent names
with tree characters or leading spaces without ambiguity:

```text
"My Project"/
├── " source files"/
│   └── main.go
├── "notes, draft.txt"
└── "├── name"/
```

Output expands every level separately, uses four columns per indentation unit,
marks every directory with `/`, and quotes names when needed. Empty directories,
types, names and sibling order survive conversion.

## Compact format

```text
TVSnake/{settings.gradle.kts,build.gradle.kts,gradle.properties,app/{build.gradle.kts,src/main/{AndroidManifest.xml,java/com/example/tvsvnake/{MainActivity.kt,GameView.kt},res/{values/strings.xml,drawable/app_banner.xml}}}}
```

Directories use `name/{children}`, commas separate children, and `/` separates
path components. `empty/` is an empty directory and `EmptyProject/` an empty root.
`name/{}` is also accepted as an empty directory. Whitespace outside quoted names
is formatting and is ignored; use quotes to preserve spaces.

```text
"My Project"/{README.md,"source files"/{main.go},"notes, draft.txt",empty/}
```

Quotes follow JSON escape rules, e.g. `"\u0645\u0644\u0641"` is an Arabic name.
A quoted component is one name, never a whole path; encoded or literal `/` and
`\` are prohibited inside names. Quotes do not bypass Windows validation (for
example a decoded double quote remains forbidden). Missing braces, unclosed
quotes, bad escapes, empty elements, extra commas and invalid paths report a
position. Nesting and document size are bounded at 256 levels and 16 MiB.

## Validation and filesystem safety

Windows naming rules are applied on all platforms for portable documents:
no absolute/UNC paths, drive letters, `.`/`..`, empty components, control/NUL
characters, `< > : " / \ | ? *`, reserved device names (including extensions),
trailing spaces/dots or case-conflicting siblings. Names are never sanitized or
silently changed. Names accepted by the text model may still exceed filesystem
length limits, which result in a clear I/O error.

All input is validated before disk access for creation. A complete plan checks
existing types and ancestor paths before writing. Existing files are kept;
new files use exclusive creation. Every existing ancestor is inspected with
`Lstat`, and Windows `FILE_ATTRIBUTE_REPARSE_POINT` covers junctions as well as
symlinks. Checks run during planning, again before execution and before each item.
No existing permissions are changed. Scan skips links/reparse points with stderr
warnings, never follows them, and refuses a linked root or ancestor.

These path checks narrow races but cannot eliminate concurrent malicious
replacement between checking a path and opening/creating it. Do not generate or
save into directories concurrently modified by untrusted processes. This version
does not use kernel handle-relative traversal and does not claim full race protection.

Scan sorts directories first, then files, using case-sensitive Go string order
(lexicographic UTF-8 bytes) within each group. Hidden names are included.
Repeated `--exclude NAME` matches the exact case-sensitive basename at any level,
with no glob expansion. Root depth is zero; directories at `--max-depth` appear
empty and stderr states that the result is depth-limited. Recreating that result
does not recreate omitted descendants. Access failures and unrepresentable names
fail the scan rather than silently producing a partial successful result.
When `--save` is inside the scanned directory its exact output path is excluded
automatically, including when the output already exists.

Clipboard is isolated behind an interface, with direct Windows Unicode APIs,
a hidden owner window, retry on busy access, and no shell evaluation of clipboard
text. Clipboard errors and an empty/nontext Clipboard fail clearly. Other OSes
support file input, scan and conversion; Clipboard is Windows-only.

## Install after publishing a release

Set `GITHUB_OWNER` and `GITHUB_REPOSITORY` in `scripts/install.ps1` and replace
those placeholders in these URLs. These URLs are templates, not current downloads.
The installer uses the native Windows architecture (amd64 or arm64), a chosen
tag or GitHub's latest release, downloads and verifies SHA-256 **before** replacing
an installation, installs into `%LOCALAPPDATA%\msk\bin`, appends only that directory
to user PATH without duplicates, and updates the invoking PowerShell session.
No Administrator access or permanent ExecutionPolicy change is needed.

Review and run the script (recommended):

```powershell
$owner = 'GITHUB_OWNER'
$repo = 'GITHUB_REPOSITORY'
Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/$owner/$repo/main/scripts/install.ps1" -OutFile install.ps1
Get-Content .\install.ps1
& .\install.ps1 -GITHUB_OWNER $owner -GITHUB_REPOSITORY $repo -Version v1.0.0
# Omit -Version to install latest; run again to update/reinstall.
```

Direct execution from the published link (only after you trust the repository):

```powershell
$owner = 'GITHUB_OWNER'; $repo = 'GITHUB_REPOSITORY'
$script = (Invoke-WebRequest -UseBasicParsing "https://raw.githubusercontent.com/$owner/$repo/main/scripts/install.ps1").Content
& ([scriptblock]::Create($script)) -GITHUB_OWNER $owner -GITHUB_REPOSITORY $repo
```

Use a tag or commit instead of `main` to pin the script. If local policy blocks
execution, an allowed process-only invocation is
`powershell -NoProfile -ExecutionPolicy Bypass -File .\install.ps1 -GITHUB_OWNER OWNER -GITHUB_REPOSITORY REPO`.
A child process updates the persistent user PATH, but cannot update its parent's
session PATH: open a new shell afterward. Corporate policies may still prohibit
script execution. The scripts support Windows PowerShell 5.1 and PowerShell 7;
older TLS/proxy environments may require administrator-managed network setup.

Checksums prove archive consistency with the published checksum manifest; they
do **not** prove publisher identity or replace a trusted code signature. Assets
and checksum manifest currently come from the same GitHub release.
Locked/running executables produce an instruction to close msk and retry.
Temporary downloads and staged replacements are cleaned up.

Uninstall using the reviewed repository script:

```powershell
& .\scripts\uninstall.ps1
```

It removes only `msk.exe`, empty installation directories, and the msk bin PATH
entry. Other files and PATH entries are kept. It does not change ExecutionPolicy.

## Build and test

Use **Go 1.26.8**, with language baseline `go 1.26.0` in `go.mod` and the exact
compiler version pinned in CI. No external modules are used, so no `go.sum` is needed.

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

Build output is `dist/windows_amd64/msk.exe` and `dist/windows_arm64/msk.exe`.
`build.ps1` restores environment settings after building and embeds version,
commit and UTC build date. Release archives contain executable, MIT license and
README at archive root:

```text
msk_v1.0.0_windows_amd64.zip
msk_v1.0.0_windows_arm64.zip
checksums.txt
```

`test-install.ps1` uses local release assets and an isolated temporary
LOCALAPPDATA, disables real PATH writes, and tests install/reinstall, checksum
failure preserving the existing binary, PATH string operations, and uninstall.
Run it in both PowerShell editions; CI does so. It does not require Pester.

Tests cover examples, Unicode, quoting, short paths, merging, ordering, invalid
input, round trips, parser fuzzing, CLI streams/codes, temporary-disk creation,
preservation, no-write preview/preflight, scan filters/depth/save exclusion and
links. Symlink/junction tests explicitly skip if platform privileges prevent setup.
Clipboard unit tests use a fake and do not disturb the user's real Clipboard.
A Windows native Clipboard test allocates a private window station and desktop,
so it never reads or modifies the user's Clipboard. It skips explicitly if Windows
denies creation of that isolated desktop. Interactive desktop Clipboard and native
ARM64 execution require their respective environments; cross compilation alone
does not establish native runtime behavior.

## Publish the first real release

1. Create your GitHub repository and copy this project into it.
2. Replace installer `GITHUB_OWNER` / `GITHUB_REPOSITORY` defaults and README URL
   placeholders with your real values; confirm the default branch in script URLs.
3. Commit and push; confirm CI succeeds on Windows and Linux, and inspect outputs.
4. Tag and push `v1.0.0`. The release workflow tests, packages both Windows
   architectures, computes `checksums.txt`, and publishes via GitHub CLI.
5. Check asset names and SHA-256 entries, test installation against that release,
   then distribute the actual raw-script link. No publication has happened locally.

CI uses read-only repository permissions. Only the tag release job gets
`contents: write` to publish; no extra scopes or secrets are required.

## Source layout and decisions

`cmd/msk` is the executable entry point. `internal/cli` handles options and sources;
`model` validates the ordered tree; `parser` parses both grammars; `formatter`
serializes them; `filesystem` plans, executes, scans and stages output; `clipboard`
contains the Windows-specific adapter. `examples` holds the matching TVSnake
documents, `scripts` builds/installs/uninstalls/tests distribution, and `.github`
holds CI and tag releases. Platform-independent logic uses only the Go standard
library. Shared directory declarations merge; duplicate files never do.

Licensed under [MIT](LICENSE).

## Verification of this implementation (2026-10-01)

Executed locally on Windows amd64 with Go 1.26.8:

- `go test -count=1 ./...`: passed all available tests. Native symlink creation
  skipped because the token lacks that privilege; junction safety passed.
- `go vet ./...`: passed.
- Parser fuzzing with round-trip assertions, 10 seconds/two workers: passed,
  109,384 executions in the recorded run.
- Release build: both Windows amd64 and arm64 executables and ZIPs created,
  with metadata and matching SHA-256 manifest.
- Actual amd64 executable: help/version, creation from both TVSnake examples,
  scan comparison, and repeat execution all passed. Both formats yielded the
  same structure, and repeat execution skipped all 20 existing project items.
- Offline installer tests passed in PowerShell 7 and Windows PowerShell 5.1:
  install/reinstall, locked-file rejection, missing/corrupt checksum rejection,
  preservation of installed binary, PATH deduplication/removal, and uninstall.

Not established locally: Linux runtime (covered by prepared CI), native ARM64
runtime, live GitHub release downloads/publication, or actual Clipboard round-trip.
The native Clipboard test skipped because Windows denied creating its private
window station, including outside the filesystem sandbox. Fake Clipboard CLI
tests passed. Real user PATH writes are deliberately disabled in installer tests;
the PATH transformation functions are tested without changing account settings.
