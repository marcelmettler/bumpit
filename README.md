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

**Unused dependency detection.** Scans your source files, scripts, and tool config files to find direct dependencies that are never referenced. Select and remove them interactively. Understands ecosystem-specific patterns so it won't flag build tools, type definitions, or platform packages that are used without being directly imported.

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

`bumpit` has two commands:

```bash
bumpit update [directory]   # show outdated packages and update them interactively
bumpit unused [directory]   # find unused direct dependencies and remove them
```

Both commands default to the current directory. Run `bumpit --help` for a full flag listing.

### `bumpit update`

```bash
bumpit update
bumpit update /path/to/project
bumpit update --show-indirect   # include indirect Go module dependencies
```

### `bumpit unused`

Scans your direct dependencies against source files, `package.json` scripts, and tool config files (`angular.json`, `nx.json`, `.eslintrc.json`, `.commitlintrc`, etc.) to find packages that are never referenced.

```bash
bumpit unused
bumpit unused /path/to/project
```

Ecosystem-aware — will not flag:
- `@types/*` packages when the corresponding package is installed
- TypeScript, ESLint plugins/configs, Nx plugins, Capacitor platform packages, Vite/jsdom (as vitest peers), Angular build tooling, commitlint configs

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

### `update` — list view

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

### `update` — detail view

| Key | Action |
|-----|--------|
| `esc` | Back to list |
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `space` | Toggle selection |
| `u` | Update this package |
| `q` | Quit |

### `unused` — list view

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `space` | Toggle selection |
| `a` | Select / deselect all visible |
| `r` | Remove selected packages |
| `q` | Quit |

## How it works

### `bumpit update`

1. **Detect** — walks the directory tree to find `package.json` and `go.mod` files
2. **Fetch outdated** — runs `pnpm outdated --json` or `go list -m -u -json all` per directory
3. **Enrich** — fetches publish date and repository URL from the npm registry for each package
4. **Changelogs** — calls the GitHub releases API, filters to the relevant version range, and renders markdown in the terminal using [glamour](https://github.com/charmbracelet/glamour)
5. **Audit** — runs `pnpm audit --json` in the background and annotates vulnerable packages
6. **Update** — runs `pnpm update --latest <pkg...>` grouped by directory for monorepo correctness

### `bumpit unused`

1. **Detect** — same file discovery as `update`
2. **Scan imports** — walks all `.js/.ts/.jsx/.tsx/.mjs/.cjs/.vue/.svelte` source files across the entire workspace root collecting `import`/`require` references
3. **Scan scripts** — checks `package.json` scripts for package name references (catches CLI tools like `prettier`)
4. **Scan configs** — reads `angular.json`, `nx.json`, `project.json`, `.eslintrc.json`, `.commitlintrc.json` for executor and parser strings
5. **Ecosystem rules** — applies per-ecosystem knowledge to skip packages that are used without being imported (build tools, type stubs, platform packages)
6. **Remove** — runs `pnpm remove <pkg...>` or `go get <mod>@none && go mod tidy` for selected packages

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

Set `BUMPIT_DEBUG=1` to write a trace log to `/tmp/bumpit-debug.log`:

```bash
BUMPIT_DEBUG=1 bumpit update
```

## License

[MIT](LICENSE)
