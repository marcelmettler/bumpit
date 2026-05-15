# chorekit

chorekit — a tidy toolkit for keeping your codebase in shape

`chorekit` brings the recurring chores of a healthy codebase — updating dependencies, trimming dead weight, auditing licenses — into a single interactive interface. No browser tabs, no context switching, no scripting one-offs.

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
chorekit update [directory]    # interactive outdated-package updater
chorekit unused [directory]    # find and remove unused direct dependencies
chorekit license [directory]   # audit dependency licenses
chorekit clean [directory]     # find and delete generated artifact directories
chorekit css [directory]       # two-way CSS audit: unused selectors and missing definitions
chorekit todo [directory]      # list all TODO, FIXME, HACK, and XXX comments
chorekit i18n [directory]      # two-way translation key audit: unused keys and undefined references
chorekit env [directory]       # two-way .env.example audit: unused vars and undefined references
chorekit audit [directory]     # security vulnerability audit with fix availability
chorekit pin [directory]       # find ^ and ~ version ranges and pin them to exact versions
chorekit sort [directory]      # sort dependency sections in package.json files alphabetically
```

All commands default to the current directory. Run `chorekit --help` for a full flag listing.

## Features

### `chorekit update`

**Changelog in the terminal.** Opens GitHub releases for the full range between your current and latest version. No browser, no tab switching.

**Breaking change detection.** Scans release notes for `BREAKING CHANGE`, `BREAKING:`, `⚠️`, and `INCOMPATIBLE`. Highlights affected packages in red.

**`minimumReleaseAge` support.** Reads your `.npmrc` for `minimum-release-age` (supports `3 days`, `72h`, `3d`). Packages that haven't aged enough show a countdown instead of a status, matching pnpm's publish safety model.

**Monorepo aware.** Recursively finds every `package.json` and `go.mod` under the project root. Skips `node_modules`, `dist`, `build`, `.git`, and similar directories. Each package shows which workspace it belongs to.

**Auto-detects package manager.** Checks for `pnpm-lock.yaml`, `yarn.lock`, and `package-lock.json` — including walking up to the workspace root.

**Security advisories.** Runs `pnpm audit` in the background and surfaces vulnerability counts next to affected packages.

**Batch updates.** Select any combination of packages and press `u`. Updates are grouped by directory and run with `pnpm update --latest`.

**Go modules.** Parses `go list -m -u -json all` output alongside npm packages in a unified list.

### `chorekit unused`

**Unused dependency detection.** Scans your source files, scripts, and tool config files to find direct dependencies that are never referenced. Select and remove them interactively. Understands ecosystem-specific patterns so it won't flag build tools, type definitions, or platform packages that are used without being directly imported.

Ecosystem-aware — will not flag:
- `@types/*` packages when the corresponding package is installed
- TypeScript, ESLint plugins/configs, Nx plugins, Capacitor platform packages, Vite/jsdom (as vitest peers), Angular build tooling, commitlint configs

### `chorekit license`

**License audit.** Reads license metadata from locally installed `node_modules` for every direct dependency. Default view shows only packages that need attention, with a plain-English explanation of what each license requires. Press `a` to see all packages.

| Indicator | Commercial use | Open-source obligation |
|-----------|---------------|------------------------|
| `✓` Permissive (MIT, ISC, Apache-2.0, BSD-*) | ✓ Free | None — keep the copyright notice; include license files when distributing |
| `⚠` Weak copyleft (LGPL-*, MPL-2.0) | ✓ Allowed | Only if you *modify the library itself* — using it as a dependency is fine |
| `✗` Strong copyleft (GPL-*, AGPL-3.0) | ✗ Not without open-sourcing | Distributing (GPL) or serving over a network (AGPL) requires open-sourcing your entire app |
| `?` Unknown — no license field | ✗ Illegal by default | All rights reserved — contact the author before using |

Handles SPDX compound expressions: `(MIT OR GPL-3.0-or-later)` is treated as permissive (consumer picks MIT). Filter with `/`, toggle sort between risk-first and alphabetical with `s`.

### `chorekit clean`

**Artifact cleanup.** Walks the project tree and finds all generated or installed directories — `node_modules`, `dist`, `.next`, `.turbo`, `coverage`, and more. Shows each with its disk size, sorted biggest-first so the most impactful targets are at the top. Select with space and press `D` to delete.

Finds: `node_modules`, `dist`, `build`, `out`, `storybook-static`, `.next`, `.nuxt`, `.output`, `.angular`, `.svelte-kit`, `.vite`, `.turbo`, `.cache`, `coverage`, `.nyc_output`.

### `chorekit css`

**Two-way CSS audit.** Checks for problems in both directions:

| Indicator | Meaning |
|-----------|---------|
| `~` (orange) | Class defined in a CSS file but never referenced in any template |
| `?` (red) | Class referenced in a template `class=` attribute but absent from every CSS file |

The `~` direction catches dead CSS you can safely delete. The `?` direction catches typos and stale references — class names that exist in markup but have no matching stylesheet rule.

Scans CSS files: `.css`, `.scss`, `.sass`, `.less`. Scans templates: `.html`, `.jsx`, `.tsx`, `.js`, `.ts`, `.vue`, `.svelte`.

**How references are detected.** Explicit `class="..."` and `className="..."` attributes drive both directions. String literals (`cx('foo')`, `clsx('bar')`, `classList.add('baz')`) are also checked but only for the `~` direction — they would produce too many false positives for the `?` direction since common English words would appear to be missing CSS definitions.

**Limitations:**
- CSS modules (`.module.css`) are skipped — class names are transformed by the bundler and referenced as `styles.className`, not as string literals.
- SCSS `&` parent selectors (e.g. `&__child`, `&:hover`) are not resolved — classes composed via `&` may appear as unused even if they are referenced.
- Tailwind utility classes are not tracked. If a Tailwind config is detected, a warning is shown.

Read-only — `chorekit css` reports findings but does not modify any files.

### `chorekit todo`

**Tech-debt overview.** Scans every source file in the project and lists all `TODO`, `FIXME`, `HACK`, and `XXX` comments in one navigable list, sorted by file and line.

| Kind | Color | Typical use |
|------|-------|-------------|
| `TODO` | blue | Planned work, missing feature |
| `FIXME` | orange | Known bug or broken code |
| `HACK` | orange | Workaround that should be cleaned up |
| `XXX` | red | Dangerous or critical tech debt |

Scans: `.go`, `.js`, `.ts`, `.jsx`, `.tsx`, `.vue`, `.svelte`, `.css`, `.scss`, `.html`, `.yaml`, `.sh`, and more. Filter with `/` to search by kind, file, or comment text. Read-only.

### `chorekit i18n`

**Two-way translation key audit.** Checks for drift between your locale files and source code in both directions:

| Indicator | Meaning |
|-----------|---------|
| `~` (orange) | Key defined in a locale file but never called in source |
| `?` (red) | `t('key')` call in source with no matching key in any locale file |

The `~` direction surfaces dead translations you can safely delete. The `?` direction catches typos and stale references — calls that will silently fall back to the raw key at runtime.

**Locale file discovery.** Scans all JSON files inside directories named `locales/`, `locale/`, `i18n/`, `translations/`, `lang/`, or `langs/`. Nested JSON objects are flattened to dot-notation keys (`section.subsection.key`). Multiple locale files are merged — a key is considered defined if it exists in any locale file.

**Source reference detection.** Matches `t('key')`, `$t('key')`, `i18n.t('key')`, and `translate('key')`. Bare `t(` calls require a non-identifier preceding character to avoid false positives from similarly-named functions. Works with Vue i18n, react-i18next, and most i18n libraries.

**Limitations:**
- Namespaced calls like `t('common:key')` (react-i18next namespaces) are matched literally — the key `common:key` must exist in the locale file as-is.
- Dynamic keys (`t(\`prefix.${var}\`)`) cannot be statically resolved and are not detected.

Read-only.

### `chorekit env`

**Two-way .env.example audit.** Checks for drift between your `.env.example` files and source code in both directions:

| Indicator | Meaning |
|-----------|---------|
| `~` (orange) | Variable defined in a `.env.example` file but never referenced in any source file |
| `?` (red) | Variable referenced in source but absent from every `.env.example` file |

The `~` direction surfaces stale env vars you can safely remove. The `?` direction catches variables used in code that have no corresponding `.env.example` entry — potential configuration gaps.

**Env file discovery.** Scans `.env.example`, `.env.sample`, `.env.template`, `.env.defaults`, and `.env.schema` files across the project. Lines starting with `#` are treated as comments; `export KEY=value` syntax is also supported.

**Source reference detection.** Matches environment variable access patterns across multiple ecosystems:
- Node.js: `process.env.KEY`, `process.env['KEY']`
- Vite / Astro: `import.meta.env.KEY`, `import.meta.env['KEY']`
- Go: `os.Getenv("KEY")`
- Python: `os.environ["KEY"]`, `os.environ.get("KEY")`
- YAML / shell: `${KEY}` substitution (docker-compose, kubernetes manifests, shell scripts)

**Limitations:**
- Shell bare `$KEY` references (without `{}`) are not detected — too noisy (`$HOME`, `$PATH`, etc. would produce false positives)
- Dynamic key construction (`process.env[varName]`) cannot be statically resolved

Read-only — `chorekit env` reports findings but does not modify any files.

### `chorekit audit`

**Security vulnerability audit.** Runs `pnpm audit` under the hood and presents the results in a compact, scannable format sorted by severity.

```
  CRIT  lodash                  <4.17.21  → >=4.17.21  Prototype Pollution
        CVE-2021-23337  ·  (direct)

  HIGH  tough-cookie            <4.1.3    → no fix      ReDoS
        CVE-2023-26136  ·  jest > jest-circus > jsdom > tough-cookie
```

Each vulnerability shows:
- **Severity badge** — `CRIT` / `HIGH` / `MOD` / `LOW` / `INFO`, color-coded
- **Vulnerable range** and **patched version** (green) or **no fix** (red) — immediate at a glance
- **Dependency path** — `(direct)` means `pnpm update <pkg>` fixes it immediately; a chain like `jest > tough-cookie` means the fix depends on a parent package releasing an update

Sorted by severity (critical first), then fixable before unfixable within each level. Filter with `/` by package name, severity, or title. Read-only.

### `chorekit pin`

**Unpinned dependency finder.** Scans every `package.json` in the project for dependencies using `^` or `~` version ranges and lets you pin them to exact versions interactively.

| Column | Example |
|--------|---------|
| Current | `^18.2.0` (orange — range) |
| Pinned | `18.2.0` (green — exact) |
| Type | `dep` / `devDep` / `peer` / `optional` |

Select with `space`, select all with `a`, then press `p` to write the exact versions back to the relevant `package.json` files. Pinning is done in-place — the file's key order and formatting are preserved; only the `^`/`~` prefix is stripped.

Covers `dependencies`, `devDependencies`, `peerDependencies`, and `optionalDependencies`.

### `chorekit sort`

**Dependency sorter.** Scans every `package.json` in the project for dependency sections whose keys are not sorted alphabetically, then sorts them interactively.

All files are pre-selected — sorting is non-destructive and safe to apply in bulk. Select with `space`, select all with `a`, then press `s` to write the sorted sections back.

Sorting is done in-place: only the affected dependency objects are rewritten. The rest of the file — `name`, `version`, `scripts`, and all other keys — is left exactly as-is, including indentation and key order.

Covers `dependencies`, `devDependencies`, `peerDependencies`, and `optionalDependencies`.

## Installation

**Download a pre-built binary** from the [releases page](https://github.com/marcelmettler/chorekit/releases/latest):

| Platform | File |
|----------|------|
| macOS (Apple Silicon) | `chorekit_darwin_arm64.tar.gz` |
| macOS (Intel) | `chorekit_darwin_amd64.tar.gz` |
| Linux x86-64 | `chorekit_linux_amd64.tar.gz` |
| Linux arm64 | `chorekit_linux_arm64.tar.gz` |
| Windows x86-64 | `chorekit_windows_amd64.zip` |

```bash
tar -xzf chorekit_darwin_arm64.tar.gz
mv chorekit /usr/local/bin/
```

**Or install with Go:**

```bash
go install github.com/marcelmettler/chorekit@latest
```

**Or build from source:**

```bash
git clone https://github.com/marcelmettler/chorekit
cd chorekit
go build -o chorekit .
```

Requires Go 1.22+. For npm packages, `pnpm` must be available in your PATH.

## Usage

### `chorekit update`

```bash
chorekit update
chorekit update /path/to/project
chorekit update --show-indirect   # include indirect Go module dependencies
```

### `chorekit unused`

```bash
chorekit unused
chorekit unused /path/to/project
```

### `chorekit license`

```bash
chorekit license
chorekit license /path/to/project
```

Reads license data from local `node_modules`. Run after `pnpm install` for accurate results. Packages that are listed in `package.json` but not yet installed are shown with a `?` indicator.

### `chorekit clean`

```bash
chorekit clean
chorekit clean /path/to/project
```

### `chorekit css`

```bash
chorekit css
chorekit css /path/to/project
```

### `chorekit todo`

```bash
chorekit todo
chorekit todo /path/to/project
```

### `chorekit i18n`

```bash
chorekit i18n
chorekit i18n /path/to/project
```

### `chorekit env`

```bash
chorekit env
chorekit env /path/to/project
```

### `chorekit audit`

```bash
chorekit audit
chorekit audit /path/to/project
```

### `chorekit pin`

```bash
chorekit pin
chorekit pin /path/to/project
```

### GitHub authentication

Changelogs are fetched from the GitHub API. Unauthenticated requests are limited to 60 per hour. `chorekit` automatically resolves credentials from your existing machine state — no setup required if you already use the GitHub CLI or have git configured with HTTPS:

1. `GITHUB_TOKEN` env var
2. `GH_TOKEN` env var
3. `gh auth token` — GitHub CLI credential store
4. `git credential fill` — system keychain (macOS Keychain, libsecret, etc.)

If none of the above are available, `chorekit` falls back to unauthenticated requests. For larger projects or private repos you can always set a token explicitly:

```bash
export GITHUB_TOKEN=ghp_...
chorekit
```

### `minimumReleaseAge`

If your `.npmrc` contains:

```ini
minimum-release-age=3 days
```

Packages published less than 3 days ago will show `⏳ Nd left` instead of a status. This matches pnpm's own behavior and gives the ecosystem time to catch regressions before you adopt a release.


## How it works

### `chorekit update`

1. **Detect** — walks the directory tree to find `package.json` and `go.mod` files
2. **Fetch outdated** — runs `pnpm outdated --json` or `go list -m -u -json all` per directory
3. **Enrich** — fetches publish date and repository URL from the npm registry for each package
4. **Changelogs** — calls the GitHub releases API, filters to the relevant version range, and renders markdown in the terminal using [glamour](https://github.com/charmbracelet/glamour)
5. **Audit** — runs `pnpm audit --json` in the background and annotates vulnerable packages
6. **Update** — runs `pnpm update --latest <pkg...>` grouped by directory for monorepo correctness

### `chorekit unused`

1. **Detect** — same file discovery as `update`
2. **Scan imports** — walks all `.js/.ts/.jsx/.tsx/.mjs/.cjs/.vue/.svelte` source files across the entire workspace root collecting `import`/`require` references
3. **Scan scripts** — checks `package.json` scripts for package name references (catches CLI tools like `prettier`)
4. **Scan configs** — reads `angular.json`, `nx.json`, `project.json`, `.eslintrc.json`, `.commitlintrc.json` for executor and parser strings
5. **Ecosystem rules** — applies per-ecosystem knowledge to skip packages that are used without being imported (build tools, type stubs, platform packages)
6. **Remove** — runs `pnpm remove <pkg...>` or `go get mod@none && go mod tidy` for selected packages

### `chorekit license`

1. **Detect** — same file discovery as `update`
2. **Read** — for each direct dependency in each `package.json`, reads the installed `node_modules/<pkg>/package.json` (checks package-local then workspace-root for hoisted deps)
3. **Categorise** — parses SPDX expressions including compound forms (`MIT OR GPL-3.0-or-later` → permissive, `MIT AND GPL-3.0` → strong copyleft); maps to one of four risk categories
4. **Deduplicate** — packages that appear in multiple `package.json` files across a monorepo are shown once
5. **Display** — default view shows only packages needing attention with plain-English action text; press `a` for the full list

### `chorekit clean`

1. **Scan** — walks the directory tree; when a directory with a known artifact name is found, records it and skips descending into it (so nested `node_modules` inside `node_modules` are not double-counted)
2. **Size** — computes total disk usage for each artifact directory
3. **Sort** — biggest directories first, so the most impactful targets are immediately visible
4. **Delete** — `D` runs `os.RemoveAll` on selected directories and reports total space freed

### `chorekit css`

1. **Pass 1 (define)** — walks all `.css`/`.scss`/`.sass`/`.less` files, strips comments and `url()` references, extracts every `.classname` selector. Skips `.module.css` files and lines containing `&`.
2. **Pass 2 (reference)** — walks all template and source files; builds two sets: `broad` (class attrs + string literals, for the `~` direction) and `explicit` (class attrs only with file/line, for the `?` direction). String literals are excluded from `explicit` to avoid false positives from common English words.
3. **Diff both ways** — `~` = defined in CSS but absent from `broad`; `?` = present in `explicit` but absent from CSS definitions
4. **Display** — combined scrollable list sorted within each direction by file then line; `/` to filter by name or file

### `chorekit todo`

1. **Scan** — walks all source files matching a broad extension list (`.go`, `.js`/`.ts`, `.vue`, `.svelte`, `.css`/`.scss`, `.html`, `.yaml`, `.sh`, and more); skips `node_modules`, `dist`, `.git`, and other artifact directories
2. **Match** — per line, applies `\b(TODO|FIXME|HACK|XXX)\b` regex; captures the text following the keyword, strips trailing comment closers (`*/`, `-->`) and `TODO(author):` prefixes
3. **Display** — read-only list sorted by file then line; color-coded by kind (`TODO` blue, `FIXME`/`HACK` orange, `XXX` red); `/` filters by kind, text, or file

### `chorekit i18n`

1. **Locale scan** — walks all JSON files inside `locales/`, `i18n/`, `translations/`, `lang/` directories; parses each with `encoding/json` and recursively flattens nested objects to dot-notation keys; deduplicates across multiple locale files
2. **Source scan** — walks `.js`/`.ts`/`.jsx`/`.tsx`/`.vue`/`.svelte`/`.html` files line by line; applies two patterns: `\$t|i18n\.t|translate` (explicit) and `[^a-zA-Z_$\d]t\s*\(` (bare `t()`) for broad coverage
3. **Diff both ways** — `~` = defined in locale but not in source refs; `?` = in source refs but absent from all locale files
4. **Display** — combined scrollable list (unused first, then undefined); `/` filters by key or file

### `chorekit env`

1. **Env file scan** — walks all `.env.example`, `.env.sample`, `.env.template`, `.env.defaults`, and `.env.schema` files; parses each line with `^(?:export\s+)?KEY=` regex; skips comments and blank lines
2. **Source scan** — walks `.js`/`.ts`/`.go`/`.py`/`.sh`/`.yaml` and related files; applies patterns for `process.env.KEY`, `import.meta.env.KEY`, `os.Getenv("KEY")`, `os.environ["KEY"]`, and `${KEY}` YAML substitution
3. **Diff both ways** — `~` = declared in `.env.example` but no matching access in source; `?` = accessed in source but absent from all env files
4. **Display** — combined scrollable list (unused first, then undefined); `/` filters by variable name or file

### `chorekit audit`

1. **Run** — executes `pnpm audit --json` from the project root; `pnpm audit` exits non-zero when vulnerabilities are found, which is treated as success (not an error)
2. **Parse** — deserialises the `advisories` map; for each advisory, collects all `findings[*].paths` into a deduplicated path list and records the installed version from the first finding
3. **Fixability** — `patched_versions` values of `""`, `"<0.0.0"`, or `"<0.0.0-0"` are treated as "no fix available"; any other value means a patch exists. `IsDirect` is set when any path contains no `>` separator
4. **Sort** — critical → high → moderate → low → info; within each severity, fixable entries appear before unfixable; then alphabetical by package name
5. **Display** — two-line compact format: severity badge, package, vulnerable range, patched range (green) or "no fix" (red), title on line 1; CVEs and shortest dependency path on line 2; `/` filters by name, severity, or title

### `chorekit pin`

1. **Scan** — walks all `package.json` files (skipping `node_modules`, `dist`, etc.); for each of the four dependency sections (`dependencies`, `devDependencies`, `peerDependencies`, `optionalDependencies`), records every entry whose version starts with `^` or `~`
2. **Display** — scrollable list showing current range (orange), exact pinned version (green), dep type, and workspace dir; select with `space`, `a` to toggle all
3. **Pin** — for selected deps, reads each `package.json` as raw bytes, replaces `"name": "^x.y.z"` → `"name": "x.y.z"` in-place using targeted string substitution to preserve file formatting and key order, then writes back

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
git clone https://github.com/marcelmettler/chorekit
cd chorekit
go build -o chorekit .
go test ./...
```

Set `CHOREKIT_DEBUG=1` to write a trace log to `/tmp/chorekit-debug.log`:

```bash
CHOREKIT_DEBUG=1 chorekit update
```

## License

[MIT](LICENSE)
