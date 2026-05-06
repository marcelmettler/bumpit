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
./bumpit [directory]
BUMPIT_DEBUG=1 ./bumpit    # enables debug log at /tmp/bumpit-debug.log

# Version
./bumpit --version
./bumpit --show-indirect    # include indirect Go module dependencies
```

## Architecture

`bumpit` is a BubblaTea TUI that shows outdated npm/Go dependencies and lets you update them interactively.

**State machine** (`internal/ui/model.go`): `stateInit → stateLoading → stateList ↔ stateDetail → stateUpdating → stateDone`. `Update()` dispatches key events through `handleKey()` → `updateList()` / `updateDetail()`.

**Data flow**:
1. `cmdDetect()` → `detect.Find()` discovers `pnpm-lock.yaml` and `go.mod` files
2. `cmdFetchPackages()` → calls `pnpm.Outdated()` / `gomod.Outdated()` per detected file
3. For npm packages: `cmdFetchRegistry()` enriches with publish date + repo URL → `cmdFetchChangelog()` fires automatically after registry data arrives
4. `cmdRunAudit()` runs `pnpm audit --json` in the background; vuln counts written onto `PackageUpdate`

**Package layout**:
- `internal/ui/` — all BubblaTea code: `model.go` (state machine), `list.go` (list view), `detail.go` (changelog detail), `styles.go` (lipgloss styles), `debug.go` (gated debug log)
- `internal/detect/` — finds `pnpm-lock.yaml` and `go.mod` files
- `internal/pkg/` — shared `PackageUpdate` struct and types; `pnpm/` and `gomod/` subdirs for each ecosystem
- `internal/changelog/` — `github.go` fetches GitHub releases/CHANGELOG; `changelog.go` orchestrates; `extract.go` parses markdown into `Highlights`
- `internal/registry/` — npm registry API for publish date and repo URL

## Key patterns

**Glamour render cache**: `PackageUpdate.CachedRender(width)` / `SetCachedRender()` avoids re-rendering markdown on every BubblaTea message (glamour takes ~4ms per render).

**Breaking change detection** (`internal/ui/detail.go`): `splitChangelogSections()` splits markdown by release headers; sections whose header contains "breaking" get a red `▌` gutter via `applyBreakingGutter()`. Individual lines are also checked by `isBreakingLineContent()` (handles `⚠️` emoji inline bullets).

**OSC 8 clickable links**: `injectMarkdownLinks()` converts `[text](url)` to OSC 8 sequences before glamour renders; `makeRenderedURLsClickable()` wraps bare URLs in the rendered output. Most terminals require Ctrl+click; Ghostty treats single-click as open.

**GitHub auth credential chain** (`internal/changelog/github.go`): `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token` → `git credential fill`. Unauthenticated requests are rate-limited to 60/hour.

**Security**: All HTTP responses are bounded — `io.LimitReader(resp.Body, 20<<20)` for GitHub, `10<<20` for npm. Package names are validated against `validPackageName` regex before being passed to `pnpm update`.

**pnpm exit code**: `pnpm outdated --json` exits 1 when there are outdated packages; the code treats exit code 1 as success and only errors on other non-zero codes.

## Release

Tagged releases (`v*`) trigger `.github/workflows/release.yml`, which cross-compiles for darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, and windows/amd64 on `ubuntu-latest` and uploads `.tar.gz`/`.zip` assets with `checksums.sha256`.

Module path: `github.com/marcelmettler/bumpit`
