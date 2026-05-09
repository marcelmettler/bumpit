# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o bumpit .
# or
./build.sh          # runs go mod tidy then builds

# Test
go test ./...
go test ./internal/ui/... -run TestDetailEsc   # single test

# Run (dev)
./bumpit update [directory]
./bumpit unused [directory]
./bumpit license [directory]
./bumpit clean [directory]
./bumpit css [directory]
./bumpit todo [directory]
BUMPIT_DEBUG=1 ./bumpit update    # enables debug log at /tmp/bumpit-debug.log

# Version / help
./bumpit --version
./bumpit update --show-indirect    # include indirect Go module dependencies
```

## Architecture

`bumpit` is a monorepo chore helper: a BubblaTea TUI with commands for updating dependencies, removing unused ones, auditing licenses, and cleaning up artifact directories. The vision is a single interactive interface for keeping a codebase healthy — dependency health, dead code removal, license compliance, and build-artifact cleanup.

**State machine** (`internal/ui/model.go`):
- `update` path: `stateInit → stateLoading → stateList ↔ stateDetail → stateUpdating → stateDone`
- `unused` path: `stateInit → stateLoading → stateUnusedList → stateRemoving → stateRemoveDone`
- `license` path: `stateInit → stateLoading → stateLicenseList`
- `clean` path: `stateInit → stateLoading → stateCleanList → stateCleanDeleting → stateCleanDone`
- `css` path: `stateLoading → stateCSSList` (bypasses `cmdDetect` — starts CSS scan directly from `Init()`)
- `todo` path: `stateLoading → stateTodoList` (bypasses `cmdDetect` — starts TODO scan directly from `Init()`)

`Update()` dispatches key events through `handleKey()` → `updateList()` / `updateDetail()` / `updateUnusedList()` / `updateLicenseList()` / `updateCleanList()` / `updateCSSList()` / `updateTodoList()`. The active path is determined by the mode flag on `Model` (`unusedMode` / `licenseMode` / `cleanMode` / `cssMode` / `todoMode`), set from `Config`.

**Data flow — `update`**:
1. `cmdDetect()` → `detect.Find()` discovers `pnpm-lock.yaml` and `go.mod` files
2. `cmdFetchPackages()` → calls `pnpm.Outdated()` / `gomod.Outdated()` per detected file
3. For npm packages: `cmdFetchRegistry()` enriches with publish date + repo URL → `cmdFetchChangelog()` fires automatically after registry data arrives
4. `cmdRunAudit()` runs `pnpm audit --json` in the background; vuln counts written onto `PackageUpdate`

**Data flow — `unused`**:
1. Same `cmdDetect()` step
2. `cmdFindUnused()` → calls `pnpm.FindUnused()` / `gomod.FindUnused()` per detected file
3. `pnpm.FindUnused()` collects references from three sources: import scanning (regex over source files), script scanning (`package.json` scripts), and config file scanning (`angular.json`, `nx.json`, `project.json`, `.eslintrc.json`, `.commitlintrc.json`). Ecosystem rules in `evalDep()` suppress false positives for build tools and platform packages.
4. `cmdRunRemove()` → `pnpm.RunRemove()` or `gomod.RunRemove()` (`go get mod@none` + `go mod tidy`)

**Data flow — `license`**:
1. Same `cmdDetect()` step
2. `cmdFindLicenses()` → calls `pnpm.FindLicenses()` per detected `package.json`; deduplicates across workspaces by package name
3. `pnpm.FindLicenses()` reads `node_modules/<pkg>/package.json` (local then root-hoisted) and extracts the `license` field, normalising string/object/array formats
4. `classifyLicense()` / `classifySPDX()` maps SPDX expressions (including `A OR B` / `A AND B` compound forms) to `LicenseCategoryStrongCopyleft` / `Unknown` / `WeakCopyleft` / `Permissive`; sort order is risky-first
5. Default view shows only non-permissive packages; press `a` to toggle full list

**Data flow — `clean`**:
1. `cmdDetect()` fires but only the root is needed; `cmdScanArtifacts()` runs immediately after
2. `clean.FindArtifacts(root)` walks the tree, records directories whose name matches a known artifact kind (`node_modules`, `dist`, `.next`, `.turbo`, `coverage`, etc.), skips descending into them, and computes each directory's size
3. Results are sorted biggest-first; user selects with space and confirms with `D`
4. `clean.Remove()` calls `os.RemoveAll` for each selected directory and reports total bytes freed

**Data flow — `css`**:
1. `Init()` detects `cssMode` and calls `cmdScanCSS()` directly — no `cmdDetect()` phase
2. `css.Scan(root)` does two passes: (a) walks `.css`/`.scss`/`.sass`/`.less` files and extracts class selectors via `\.([a-zA-Z][\w-]*)` after stripping comments and `url()` content; (b) walks template/source files and builds two reference sets: `broad` (class attrs + string literals) and `explicit` (class attrs only with file/line location)
3. `.module.css` files are skipped (CSS modules obfuscate class names); lines containing `&` are skipped (SCSS parent selectors can't be statically resolved)
4. `ScanResult.Unused` = defined in CSS but absent from `broad`; `ScanResult.Undefined` = present in `explicit` but absent from CSS definitions
5. Both sets merged into `cssItems []cssItem` in the model; `cssItem.undefined` flag distinguishes direction; rendered with `~` (orange, unused) and `?` (red, undefined) indicators in a single scrollable list

**Data flow — `todo`**:
1. `Init()` detects `todoMode` and calls `cmdScanTodo()` directly — no `cmdDetect()` phase
2. `todo.Scan(root)` walks the tree skipping the same artifact directories as `clean`/`css`; scans a broad set of source extensions
3. Each file is read line by line; `reKeyword` matches `TODO|FIXME|HACK|XXX`; trailing comment closers and `TODO(author):` patterns are stripped from the captured text
4. Results sorted by file then line; `rebuildTodoFiltered()` applies the filter query across kind, text, and file path

**Package layout**:
- `internal/ui/` — all BubblaTea code: `model.go` (state machine), `list.go` (update list view), `detail.go` (changelog detail), `unused.go` (unused list view + remove summary), `license.go` (license audit view), `clean.go` (clean workspace view), `css.go` (CSS unused-class view), `todo.go` (TODO audit view), `loading.go` (indeterminate progress bar), `styles.go` (lipgloss styles), `debug.go` (gated debug log)
- `internal/detect/` — finds `pnpm-lock.yaml` and `go.mod` files
- `internal/pkg/` — shared structs (`PackageUpdate`, `UnusedPackage`, `LicenseInfo`, `LicenseCategory`, `ArtifactDir`, `CSSClass`, `TodoItem`); `pnpm/`, `gomod/`, `clean/`, `css/`, and `todo/` subdirs for each domain
- `internal/changelog/` — `github.go` fetches GitHub releases/CHANGELOG; `changelog.go` orchestrates; `extract.go` parses markdown into `Highlights`
- `internal/registry/` — npm registry API for publish date and repo URL

## Key patterns

**Glamour render cache**: `PackageUpdate.CachedRender(width)` / `SetCachedRender()` avoids re-rendering markdown on every BubblaTea message (glamour takes ~4ms per render).

**Breaking change detection** (`internal/ui/detail.go`): `splitChangelogSections()` splits markdown by release headers; sections whose header contains "breaking" get a red `▌` gutter via `applyBreakingGutter()`. Individual lines are also checked by `isBreakingLineContent()` (handles `⚠️` emoji inline bullets).

**OSC 8 clickable links**: `injectMarkdownLinks()` converts `[text](url)` to OSC 8 sequences before glamour renders; `makeRenderedURLsClickable()` wraps bare URLs in the rendered output. Most terminals require Ctrl+click; Ghostty treats single-click as open.

**GitHub auth credential chain** (`internal/changelog/github.go`): `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token` → `git credential fill`. Unauthenticated requests are rate-limited to 60/hour.

**Security**: All HTTP responses are bounded — `io.LimitReader(resp.Body, 20<<20)` for GitHub, `10<<20` for npm. Package names are validated against `validPackageName` regex before being passed to `pnpm update`.

**pnpm exit code**: `pnpm outdated --json` exits 1 when there are outdated packages; the code treats exit code 1 as success and only errors on other non-zero codes.

**Unused detection ecosystem rules** (`internal/pkg/pnpm/unused.go`): `evalDep()` checks packages against a `depCtx` that carries boolean flags (`isTS`, `hasEslint`, `hasNx`, `hasCapacitor`, `hasVitest`, `hasAngular`, `hasCommitlint`) derived from `allDeps`. Each flag enables a corresponding `is*EcosystemPackage()` predicate. `collectToolConfigRefs()` scans `angular.json`, `nx.json`, `project.json`, `.eslintrc.json`, `.commitlintrc.json` for executor/parser strings to mark packages as referenced before the ecosystem checks run. To add a new ecosystem: add a flag to `depCtx`, set it in `FindUnused`, add a predicate function, and add a guard block in `evalDep`.

**License classification** (`internal/pkg/pnpm/licenses.go`): `classifyLicense()` delegates to `classifySPDX()`, which recursively evaluates SPDX compound expressions. `splitSPDXOp()` splits at the top level only (respects parentheses). OR expressions take the most permissive branch (highest `LicenseCategory` int); AND expressions take the most restrictive (lowest). Leaf identifiers are looked up in three static maps after stripping `-only`/`-or-later`/`+` suffixes and `WITH <exception>` clauses. `LicenseCategory` int ordering: 0=StrongCopyleft, 1=Unknown, 2=WeakCopyleft, 3=Permissive — higher is safer, used directly for sort-ascending-risky-first. To add a license: update the appropriate map.

**Artifact scanning** (`internal/pkg/clean/clean.go`): `FindArtifacts()` walks the tree with `filepath.WalkDir`, returns `fs.SkipDir` when it hits a known artifact directory name so it never descends into `node_modules` etc. `dirSize()` walks the artifact dir separately to compute total bytes. `FormatSize()` formats bytes to human-readable GB/MB/KB. `Remove()` calls `os.RemoveAll` sequentially and accumulates bytes freed.

**TODO scanning** (`internal/pkg/todo/todo.go`): `Scan(root)` walks all source files matching a broad extension set, reads each file line by line with `bufio.Scanner`, and applies `\b(TODO|FIXME|HACK|XXX)\b` to each line. Captures the trailing text, strips comment closers (`*/`, `-->`) and `TODO(author):` prefix patterns. Results sorted by file then line. `internal/ui/todo.go` renders a color-coded list (`TODO` blue, `FIXME`/`HACK` orange, `XXX` red); filter searches kind, text, and file path.

**CSS class scanning** (`internal/pkg/css/css.go`): `Scan(root)` performs two `filepath.WalkDir` passes. Pass 1 extracts CSS class definitions. Pass 2 calls `extractClassRefs(content, relPath)` which returns two maps: `broad` (class attrs + string literals, used for `ScanResult.Unused`) and `explicit` (class attrs only with file/line, used for `ScanResult.Undefined`). String literals are excluded from `explicit` to avoid false positives — common English words would otherwise appear as missing CSS definitions. `.module.css` files and lines containing `&` are skipped. `detectTailwind()` checks for tailwind config files at root. The UI merges both result sets into `[]cssItem` with an `undefined bool` flag; `~` = unused, `?` = undefined.

**Adding a new command**: (1) add a `*Mode bool` to `Config` and `Model`, (2) add new `state*` constants, (3) add `msg*Done` message types, (4) add `cmd*` methods, (5) if it needs package-file detection add a branch in `msgDetectDone`; if not (like `css`) override `Init()` to skip `cmdDetect`, (6) handle messages in `Update()`, (7) add a key handler method, (8) add render calls in `View()`, (9) add render functions in a new `internal/ui/<command>.go`, (10) wire up in `main.go`.

## Release

Tagged releases (`v*`) trigger `.github/workflows/release.yml`, which cross-compiles for darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, and windows/amd64 on `ubuntu-latest` and uploads `.tar.gz`/`.zip` assets with `checksums.sha256`.

Module path: `github.com/marcelmettler/bumpit`
