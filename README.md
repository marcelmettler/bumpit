# bumpit

A terminal UI for keeping your monorepo in shape.

`bumpit` brings the recurring chores of a healthy codebase — updating dependencies, trimming dead weight, auditing licenses — into a single interactive interface. No browser tabs, no context switching, no scripting one-offs.

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

## Commands

```bash
bumpit update [directory]    # interactive outdated-package updater
bumpit unused [directory]    # find and remove unused direct dependencies
bumpit license [directory]   # audit dependency licenses
bumpit clean [directory]     # find and delete generated artifact directories
```

All commands default to the current directory. Run `bumpit --help` for a full flag listing.

## Features

### `bumpit update`

**Changelog in the terminal.** Opens GitHub releases for the full range between your current and latest version. No browser, no tab switching.

**Breaking change detection.** Scans release notes for `BREAKING CHANGE`, `BREAKING:`, `⚠️`, and `INCOMPATIBLE`. Highlights affected packages in red.

**`minimumReleaseAge` support.** Reads your `.npmrc` for `minimum-release-age` (supports `3 days`, `72h`, `3d`). Packages that haven't aged enough show a countdown instead of a status, matching pnpm's publish safety model.

**Monorepo aware.** Recursively finds every `package.json` and `go.mod` under the project root. Skips `node_modules`, `dist`, `build`, `.git`, and similar directories. Each package shows which workspace it belongs to.

**Auto-detects package manager.** Checks for `pnpm-lock.yaml`, `yarn.lock`, and `package-lock.json` — including walking up to the workspace root.

**Security advisories.** Runs `pnpm audit` in the background and surfaces vulnerability counts next to affected packages.

**Batch updates.** Select any combination of packages and press `u`. Updates are grouped by directory and run with `pnpm update --latest`.

**Go modules.** Parses `go list -m -u -json all` output alongside npm packages in a unified list.

### `bumpit unused`

**Unused dependency detection.** Scans your source files, scripts, and tool config files to find direct dependencies that are never referenced. Select and remove them interactively. Understands ecosystem-specific patterns so it won't flag build tools, type definitions, or platform packages that are used without being directly imported.

Ecosystem-aware — will not flag:
- `@types/*` packages when the corresponding package is installed
- TypeScript, ESLint plugins/configs, Nx plugins, Capacitor platform packages, Vite/jsdom (as vitest peers), Angular build tooling, commitlint configs

### `bumpit license`

**License audit.** Reads license metadata from locally installed `node_modules` for every direct dependency. Default view shows only packages that need attention, with a plain-English explanation of what each license requires. Press `a` to see all packages.

| Indicator | Commercial use | Open-source obligation |
|-----------|---------------|------------------------|
| `✓` Permissive (MIT, ISC, Apache-2.0, BSD-*) | ✓ Free | None — keep the copyright notice; include license files when distributing |
| `⚠` Weak copyleft (LGPL-*, MPL-2.0) | ✓ Allowed | Only if you *modify the library itself* — using it as a dependency is fine |
| `✗` Strong copyleft (GPL-*, AGPL-3.0) | ✗ Not without open-sourcing | Distributing (GPL) or serving over a network (AGPL) requires open-sourcing your entire app |
| `?` Unknown — no license field | ✗ Illegal by default | All rights reserved — contact the author before using |

Handles SPDX compound expressions: `(MIT OR GPL-3.0-or-later)` is treated as permissive (consumer picks MIT). Filter with `/`, toggle sort between risk-first and alphabetical with `s`.

### `bumpit clean`

**Artifact cleanup.** Walks the project tree and finds all generated or installed directories — `node_modules`, `dist`, `.next`, `.turbo`, `coverage`, and more. Shows each with its disk size, sorted biggest-first so the most impactful targets are at the top. Select with space and press `D` to delete.

Finds: `node_modules`, `dist`, `build`, `out`, `storybook-static`, `.next`, `.nuxt`, `.output`, `.angular`, `.svelte-kit`, `.vite`, `.turbo`, `.cache`, `coverage`, `.nyc_output`.

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

### `bumpit update`

```bash
bumpit update
bumpit update /path/to/project
bumpit update --show-indirect   # include indirect Go module dependencies
```

### `bumpit unused`

```bash
bumpit unused
bumpit unused /path/to/project
```

### `bumpit license`

```bash
bumpit license
bumpit license /path/to/project
```

Reads license data from local `node_modules`. Run after `pnpm install` for accurate results. Packages that are listed in `package.json` but not yet installed are shown with a `?` indicator.

### `bumpit clean`

```bash
bumpit clean
bumpit clean /path/to/project
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
6. **Remove** — runs `pnpm remove <pkg...>` or `go get mod@none && go mod tidy` for selected packages

### `bumpit license`

1. **Detect** — same file discovery as `update`
2. **Read** — for each direct dependency in each `package.json`, reads the installed `node_modules/<pkg>/package.json` (checks package-local then workspace-root for hoisted deps)
3. **Categorise** — parses SPDX expressions including compound forms (`MIT OR GPL-3.0-or-later` → permissive, `MIT AND GPL-3.0` → strong copyleft); maps to one of four risk categories
4. **Deduplicate** — packages that appear in multiple `package.json` files across a monorepo are shown once
5. **Display** — default view shows only packages needing attention with plain-English action text; press `a` for the full list

### `bumpit clean`

1. **Scan** — walks the directory tree; when a directory with a known artifact name is found, records it and skips descending into it (so nested `node_modules` inside `node_modules` are not double-counted)
2. **Size** — computes total disk usage for each artifact directory
3. **Sort** — biggest directories first, so the most impactful targets are immediately visible
4. **Delete** — `D` runs `os.RemoveAll` on selected directories and reports total space freed

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
