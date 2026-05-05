# bumpit

An interactive TUI for updating dependencies — without leaving the terminal.

Instead of running `pnpm outdated`, opening ten GitHub changelog pages, and guessing which updates are safe, `bumpit` gives you a single interface: a searchable, sortable list of outdated packages with changelogs rendered inline, breaking-change warnings, and multi-select batch updates.

```
  Package Updater
  15 packages  Sort: update type (MAJOR first)  [s to cycle]

    Package                  Current      Latest   Kind    Status          Dir/Type
  ▶ [x] react                  18.2.0  →  19.0.0   MAJOR   ⚠ Breaking     (web) dep
    [ ] tailwindcss             3.4.1   →   4.0.0   MAJOR   ⚠ Breaking     (root) dep
    [ ] typescript              5.3.3   →   5.8.3   minor   ✓ OK           (root) dev
    [ ] @types/node            20.11.0  →  22.0.0   MAJOR   ⏳ 1d left     (root) dev
    [x] axios                   1.6.8   →   1.7.9   minor   ✓ OK           (api) dep
    [ ] lodash                 4.17.20  →  4.17.21  patch   ✓ OK           (root) dep

  2 package(s) selected

  j/k: navigate  space: select  a: all  enter: detail  /: filter  u: update  s: sort  ?: help  q: quit
```

## Features

**Changelog in the terminal.** Opens GitHub releases for the full range between your current and latest version. No browser, no tab switching.

**Breaking change detection.** Scans release notes for `BREAKING CHANGE`, `BREAKING:`, `⚠️`, and `INCOMPATIBLE`. Highlights affected packages in red.

**`minimumReleaseAge` support.** Reads your `.npmrc` for `minimum-release-age` (supports `3 days`, `72h`, `3d`). Packages that haven't aged enough show a countdown instead of a status, matching pnpm's publish safety model.

**Monorepo aware.** Recursively finds every `package.json` and `go.mod` under the project root. Skips `node_modules`, `dist`, `build`, `.git`, and similar directories. Each package shows which workspace it belongs to.

**Auto-detects package manager.** Checks for `pnpm-lock.yaml`, `yarn.lock`, and `package-lock.json` — including walking up to the workspace root.

**Security advisories.** Runs `pnpm audit` in the background and surfaces vulnerability counts next to affected packages.

**Batch updates.** Select any combination of packages and press `u`. Updates are grouped by directory and run with `pnpm update --latest`.

**Go modules.** Parses `go list -m -u -json all` output alongside npm packages in a unified list.

## Installation

**Download a pre-built binary** from the [releases page](https://github.com/marcelmettler/bumpit/releases/latest):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `bumpit_darwin_arm64.tar.gz` |
| macOS (Intel) | `bumpit_darwin_amd64.tar.gz` |
| Linux x86-64 | `bumpit_linux_amd64.tar.gz` |
| Linux arm64 | `bumpit_linux_arm64.tar.gz` |
| Windows x86-64 | `bumpit_windows_amd64.zip` |

```bash
tar -xzf bumpit_darwin_arm64.tar.gz
mv bumpit /usr/local/bin/
```

**Or install with Go:**

```bash
go install github.com/marcelmettler/bumpit@latest
```

**Or build from source:**

```bash
git clone https://github.com/marcelmettler/bumpit
cd bumpit
go build -o bumpit .
```

Requires Go 1.22+. For npm packages, `pnpm` must be available in your PATH.

## Usage

Run from any project directory:

```bash
bumpit
```

Or point it at a specific path:

```bash
bumpit /path/to/project
```

### GitHub authentication

Changelogs are fetched from the GitHub API. Unauthenticated requests are limited to 60 per hour. `bumpit` automatically resolves credentials from your existing machine state — no setup required if you already use the GitHub CLI or have git configured with HTTPS:

1. `GITHUB_TOKEN` env var
2. `GH_TOKEN` env var
3. `gh auth token` — GitHub CLI credential store
4. `git credential fill` — system keychain (macOS Keychain, libsecret, etc.)

If none of the above are available, `bumpit` falls back to unauthenticated requests. For larger projects or private repos you can always set a token explicitly:

```bash
export GITHUB_TOKEN=ghp_...
bumpit
```

### `minimumReleaseAge`

If your `.npmrc` contains:

```ini
minimum-release-age=3 days
```

Packages published less than 3 days ago will show `⏳ Nd left` instead of a status. This matches pnpm's own behavior and gives the ecosystem time to catch regressions before you adopt a release.

## Key bindings

### List view

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `space` | Toggle selection |
| `a` | Select / deselect all visible |
| `enter` | Open changelog detail |
| `/` | Filter by name |
| `s` | Cycle sort order (update type → name → age) |
| `u` | Update selected packages |
| `?` | Toggle help overlay |
| `q` | Quit |

### Detail view

| Key | Action |
|-----|--------|
| `esc` | Back to list |
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `space` | Toggle selection |
| `u` | Update this package |
| `q` | Quit |

## How it works

1. **Detect** — walks the directory tree to find `package.json` and `go.mod` files
2. **Fetch outdated** — runs `pnpm outdated --json` or `go list -m -u -json all` per directory
3. **Enrich** — fetches publish date and repository URL from the npm registry for each package
4. **Changelogs** — calls the GitHub releases API, filters to the relevant version range, and renders markdown in the terminal using [glamour](https://github.com/charmbracelet/glamour)
5. **Audit** — runs `pnpm audit --json` in the background and annotates vulnerable packages
6. **Update** — runs `pnpm update --latest <pkg...>` grouped by directory for monorepo correctness

## Status indicators

| Indicator | Meaning |
|-----------|---------|
| `⚠ Breaking` | Release notes contain breaking change markers |
| `⏳ Nd left` | Published too recently — below `minimumReleaseAge` |
| `✓ OK` | No breaking changes detected, age requirement met |
| `…` | Changelog is still loading |

## Update kinds

| Badge | Meaning |
|-------|---------|
| `MAJOR` (red) | Breaking semver bump — review carefully |
| `minor` (yellow) | Backwards-compatible new features |
| `patch` (green) | Bug fixes only |

## Contributing

Bug reports and pull requests are welcome. For significant changes, open an issue first to discuss the approach.

```bash
git clone https://github.com/marcelmettler/bumpit
cd bumpit
go build -o bumpit .
go test ./...
```

Set `BUMPIT_DEBUG=1` to write a trace log to `/tmp/bumpit-debug.log`.

## License

[MIT](LICENSE)
