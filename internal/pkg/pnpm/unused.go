package pnpm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/marcelmettler/bumpit/internal/detect"
	"github.com/marcelmettler/bumpit/internal/pkg"
)

// importRegex matches ES module imports, CJS require, and dynamic imports of bare package specifiers.
// Excludes relative paths (./  ../) and absolute paths (/).
var importRegex = regexp.MustCompile(`(?:from|require|import)\s*\(?['"]([^'"./][^'"]*?)['"]`)

// scssUseRegex matches SCSS @use and @import directives.
// Handles the webpack-style ~ prefix for node_modules references.
var scssUseRegex = regexp.MustCompile(`@(?:use|import)\s+['"]~?([^'"]+)['"]`)

// jsonStringRegex extracts all double-quoted string values from a JSON file.
var jsonStringRegex = regexp.MustCompile(`"([^"\\]+)"`)

// eslintStringRegex extracts both single and double quoted strings from ESLint
// config files (which may be JSON, JS, or TS format).
var eslintStringRegex = regexp.MustCompile(`['"]([^'"]+)['"]`)

type packageJSONDeps struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

// WorkspaceRefs holds pre-scanned data from a single workspace walk,
// shared across multiple FindUnused calls in a monorepo.
type WorkspaceRefs struct {
	// All bare package names referenced in source files and tool config files.
	refs map[string]bool
	// Concatenated text of all .husky/* files for substring matching.
	huskyText string
}

// ScanWorkspace walks root once and collects all package references from
// source files, SCSS files, and tool config files. The result is passed to
// FindUnused so that monorepos only pay the walk cost once per workspace.
func ScanWorkspace(root string) *WorkspaceRefs {
	ws := &WorkspaceRefs{refs: make(map[string]bool)}

	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "dist", "build", ".next", ".nuxt",
				"vendor", "coverage", ".turbo", ".cache", "out", ".output":
				return fs.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		switch filepath.Ext(path) {
		case ".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs", ".vue", ".svelte":
			for _, match := range importRegex.FindAllSubmatch(content, -1) {
				if len(match) > 1 {
					ws.refs[packageNameFromImport(string(match[1]))] = true
				}
			}
		case ".scss", ".sass", ".css":
			for _, match := range scssUseRegex.FindAllSubmatch(content, -1) {
				if len(match) > 1 {
					ref := string(match[1])
					if !strings.HasPrefix(ref, ".") && !strings.HasPrefix(ref, "/") {
						ws.refs[packageNameFromImport(ref)] = true
					}
				}
			}
		}
		if isEslintConfigFile(d.Name()) {
			addEslintConfigRefs(content, ws.refs)
		} else if isToolConfigFile(d.Name()) {
			for _, match := range jsonStringRegex.FindAllSubmatch(content, -1) {
				if len(match) >= 2 {
					ws.refs[packageNameFromExecutor(string(match[1]))] = true
				}
			}
		}
		return nil
	})

	huskyDir := filepath.Join(root, ".husky")
	var huskyBuf strings.Builder
	_ = filepath.WalkDir(huskyDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		huskyBuf.Write(content)
		return nil
	})
	ws.huskyText = huskyBuf.String()

	return ws
}

// depCtx bundles the context needed to evaluate a single dependency.
type depCtx struct {
	allDeps       map[string]bool
	used          map[string]bool // imports + script refs + config refs
	isTS          bool
	hasEslint            bool
	hasAngularEslint     bool
	hasTypescriptEslint  bool
	hasNx                bool
	hasCapacitor  bool
	hasVitest     bool
	hasAngular    bool
	hasCommitlint     bool
	hasGraphqlCodegen bool
	dir               string
	dirName       string
}

// FindUnused scans dir for npm packages listed in package.json that are never
// referenced in source files, scripts, or tool-specific config files.
// ws must be obtained by calling ScanWorkspace(root) once before iterating
// over multiple package.json files — this avoids redundant workspace walks.
func FindUnused(dir, root string, ws *WorkspaceRefs) ([]*pkg.UnusedPackage, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var pkgJSON packageJSONDeps
	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	allDeps := make(map[string]bool, len(pkgJSON.Dependencies)+len(pkgJSON.DevDependencies))
	for name := range pkgJSON.Dependencies {
		allDeps[name] = true
	}
	for name := range pkgJSON.DevDependencies {
		allDeps[name] = true
	}

	// Seed used from pre-scanned workspace refs (imports + config files).
	used := make(map[string]bool, len(ws.refs))
	for k := range ws.refs {
		used[k] = true
	}

	// Script refs are per-package (depend on each package.json's scripts field).
	for k := range collectScriptRefs(pkgJSON.Scripts, allDeps) {
		used[k] = true
	}

	// Husky refs: check raw pre-read text for this package's dep names.
	for name := range allDeps {
		if strings.Contains(ws.huskyText, name) {
			used[name] = true
		}
	}

	ctx := depCtx{
		allDeps:       allDeps,
		used:          used,
		isTS:          allDeps["typescript"],
		hasEslint:           allDeps["eslint"],
		hasAngularEslint:    hasPrefixInDeps(allDeps, "@angular-eslint/") || allDeps["angular-eslint"],
		hasTypescriptEslint: hasPrefixInDeps(allDeps, "@typescript-eslint/") || allDeps["typescript-eslint"],
		hasNx:               allDeps["nx"],
		hasCapacitor:  allDeps["@capacitor/core"],
		hasVitest:     allDeps["vitest"],
		hasAngular:    allDeps["@angular/core"],
		hasCommitlint:     hasPrefixInDeps(allDeps, "@commitlint/"),
		hasGraphqlCodegen: allDeps["@graphql-codegen/cli"],
		dir:           dir,
		dirName:       detect.ShortName(dir, root),
	}

	var unused []*pkg.UnusedPackage
	for name := range pkgJSON.Dependencies {
		if u := evalDep(name, pkg.DepDependencies, ctx); u != nil {
			unused = append(unused, u)
		}
	}
	for name := range pkgJSON.DevDependencies {
		if u := evalDep(name, pkg.DepDevDependencies, ctx); u != nil {
			unused = append(unused, u)
		}
	}
	return unused, nil
}

// evalDep decides whether a single dependency should be reported as unused.
func evalDep(name string, depType pkg.DepType, ctx depCtx) *pkg.UnusedPackage {
	// TypeScript compiler, runtime helpers, and SWC build backend are invoked by
	// the build system, not imported in application source.
	if ctx.isTS && isTypescriptBuildPackage(name) {
		return nil
	}

	// @types/* packages are never imported directly. Skip only when the base
	// package is installed AND actively used — if the base is also flagged as
	// unused, flag the @types package too.
	if strings.HasPrefix(name, "@types/") {
		base := atTypesBaseName(name)
		if isRuntimeName(base) {
			return nil
		}
		if ctx.allDeps[base] && ctx.used[packageNameFromImport(base)] {
			return nil
		}
		return &pkg.UnusedPackage{Name: name, Dir: ctx.dir, DirName: ctx.dirName, Source: "npm", DepType: depType}
	}

	// ESLint plugins and configs use shorthand names in .eslintrc that don't
	// match their npm package names, so skip them when eslint is present.
	if ctx.hasEslint && isEslintEcosystemPackage(name) {
		return nil
	}

	// angular-eslint / @angular-eslint/* — self-referential scope skip.
	// "angular-eslint" is the meta-package; "@angular-eslint/*" are the scoped
	// sub-packages. Rules like '@angular-eslint/template/...' can't be
	// reverse-mapped to package names, so skip the whole family.
	if ctx.hasAngularEslint && isAngularEslintPackage(name) {
		return nil
	}

	// @typescript-eslint/* packages and the typescript-eslint meta-package are
	// interconnected: @typescript-eslint/utils and @typescript-eslint/type-utils
	// are internal utilities that no config file ever references directly.
	// The enhanced ESLint config scanner catches the parser and plugin, but utils
	// and similar internal packages require the self-referential scope skip.
	if ctx.hasTypescriptEslint && isTypescriptEslintPackage(name) {
		return nil
	}

	// Nx plugins are referenced in nx.json / project.json executors, not imports.
	if ctx.hasNx && isNxEcosystemPackage(name) {
		return nil
	}

	// Capacitor platform packages (@capacitor/ios, @capacitor/android, etc.) are
	// synced by the Capacitor CLI, never imported in source. Imported plugins are
	// already in ctx.used and are never reached by this check.
	if ctx.hasCapacitor && isCapacitorEcosystemPackage(name) {
		return nil
	}

	// vite is a required peer dep of vitest and is not imported directly when
	// using 'vitest/config'. jsdom and happy-dom are test environments configured
	// as strings in vitest/jest config, not imported.
	if ctx.hasVitest && isVitestEcosystemPackage(name) {
		return nil
	}

	// Angular build infrastructure is invoked by the Angular CLI / build system,
	// not imported in application source. Builders are referenced in angular.json.
	if ctx.hasAngular && isAngularBuildPackage(name) {
		return nil
	}

	// commitlint configs are referenced in .commitlintrc files as extend strings,
	// not imported in source code.
	if ctx.hasCommitlint && isCommitlintPackage(name) {
		return nil
	}

	// GraphQL codegen plugins are referenced as preset/plugin strings in codegen.yml
	// (e.g. preset: 'near-operation-file' → @graphql-codegen/near-operation-file).
	// They are never imported in application source.
	if ctx.hasGraphqlCodegen && isGraphqlCodegenPackage(name) {
		return nil
	}

	if ctx.used[packageNameFromImport(name)] {
		return nil
	}
	return &pkg.UnusedPackage{Name: name, Dir: ctx.dir, DirName: ctx.dirName, Source: "npm", DepType: depType}
}

// cliAliases maps npm package names whose CLI binary differs from the package name.
// collectScriptRefs uses these to detect script usage when the package name itself
// never appears in scripts (e.g. "firebase deploy" → firebase-tools).
var cliAliases = map[string]string{
	"firebase-tools":      "firebase",
	"@graphql-codegen/cli": "graphql-codegen",
}

// collectScriptRefs returns installed packages whose names appear in any package.json script.
// This catches CLI tools like prettier, tsc, jest, vitest that are run but never imported.
func collectScriptRefs(scripts map[string]string, allDeps map[string]bool) map[string]bool {
	refs := make(map[string]bool)
	for _, script := range scripts {
		for name := range allDeps {
			if strings.Contains(script, name) {
				refs[name] = true
				continue
			}
			if alias, ok := cliAliases[name]; ok && strings.Contains(script, alias) {
				refs[name] = true
			}
		}
	}
	return refs
}

// isEslintConfigFile reports whether a filename is an ESLint config file in any
// of the supported formats (legacy JSON/JS and flat config).
func isEslintConfigFile(name string) bool {
	switch name {
	case ".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs",
		".eslintrc.yaml", ".eslintrc.yml",
		"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs",
		"eslint.config.ts", "eslint.config.mts":
		return true
	}
	return false
}

// addEslintConfigRefs extracts package references from an ESLint config file and
// adds them to refs. Handles both JSON and JS/TS formats (single + double quotes)
// and expands ESLint-specific shorthand notation:
//
//	"plugin:@scope/config" → @scope/eslint-plugin
//	"plugin:name/config"   → eslint-plugin-name
//	"@scope"               → @scope/eslint-plugin (plugins array shorthand)
//	"@scope/pkg"           → @scope/pkg (direct reference, e.g. parser)
func addEslintConfigRefs(content []byte, refs map[string]bool) {
	for _, match := range eslintStringRegex.FindAllSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		s := string(match[1])
		// Skip built-ins, relative/absolute paths, and strings with whitespace.
		if s == "" || strings.ContainsAny(s, " \t\n") ||
			strings.HasPrefix(s, "eslint:") || strings.HasPrefix(s, "next:") ||
			strings.HasPrefix(s, ".") || strings.HasPrefix(s, "/") {
			continue
		}

		// "plugin:@scope/config" or "plugin:name/config" → expand to the plugin package.
		if strings.HasPrefix(s, "plugin:") {
			rest := strings.TrimPrefix(s, "plugin:")
			if idx := strings.Index(rest, "/"); idx != -1 {
				base := rest[:idx]
				if strings.HasPrefix(base, "@") {
					refs[base+"/eslint-plugin"] = true
				} else {
					refs["eslint-plugin-"+base] = true
				}
			}
			continue
		}

		// Add the raw string as a direct package name reference.
		pkg := packageNameFromImport(s)
		if pkg != "" {
			refs[pkg] = true
		}

		// "@scope" without a sub-path is a plugin shorthand in a plugins array
		// (e.g. "@typescript-eslint" → @typescript-eslint/eslint-plugin).
		if strings.HasPrefix(s, "@") && !strings.Contains(s[1:], "/") {
			refs[s+"/eslint-plugin"] = true
		}
	}
}

// isToolConfigFile reports whether a filename is a known tool config file whose
// string values may reference npm package names.
func isToolConfigFile(name string) bool {
	switch name {
	case "nx.json", "project.json",
		"angular.json",
		".commitlintrc.json", ".commitlintrc",
		"jest.config.json":
		return true
	}
	return false
}

// packageNameFromExecutor extracts the npm package from a tool executor string.
// "@nx/js:build" → "@nx/js", "@angular-devkit/build-angular:application" → "@angular-devkit/build-angular".
func packageNameFromExecutor(s string) string {
	if idx := strings.IndexByte(s, ':'); idx != -1 {
		s = s[:idx]
	}
	return packageNameFromImport(s)
}

// packageNameFromImport extracts the bare package name from an import path.
// "@scope/pkg/sub" → "@scope/pkg", "lodash/fp" → "lodash".
func packageNameFromImport(path string) string {
	if strings.HasPrefix(path, "@") {
		parts := strings.SplitN(path, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return path
	}
	return strings.SplitN(path, "/", 2)[0]
}

// atTypesBaseName converts a @types package name to its corresponding runtime package name.
// "@types/react" → "react", "@types/babel__core" → "@babel/core".
func atTypesBaseName(name string) string {
	base := strings.TrimPrefix(name, "@types/")
	// DefinitelyTyped scoped package convention: @types/scope__pkg → @scope/pkg
	if idx := strings.Index(base, "__"); idx != -1 {
		return "@" + base[:idx] + "/" + base[idx+2:]
	}
	return base
}

// isRuntimeName reports whether name is a JS runtime that would never appear in package.json deps.
func isRuntimeName(name string) bool {
	switch name {
	case "node", "bun", "deno":
		return true
	}
	return false
}

// isTypescriptBuildPackage reports whether a package is part of the TypeScript / SWC
// build toolchain. These are invoked by compilers and build systems, not imported in source.
func isTypescriptBuildPackage(name string) bool {
	switch name {
	case "typescript", "tslib", "ts-node", "ts-patch":
		return true
	}
	return strings.HasPrefix(name, "@swc/") || strings.HasPrefix(name, "@swc-node/")
}

// isEslintEcosystemPackage reports whether a package belongs to the ESLint ecosystem.
// ESLint plugins and configs use shorthand names in config files that don't match
// their npm package names, so they can't be detected by import scanning.
func isEslintEcosystemPackage(name string) bool {
	if name == "eslint" {
		return true
	}
	if strings.HasPrefix(name, "eslint-plugin-") || strings.HasPrefix(name, "eslint-config-") {
		return true
	}
	// Scoped eslint packages: @scope/eslint-plugin, @scope/eslint-config, @scope/eslint-plugin-*
	base := name
	if strings.HasPrefix(name, "@") {
		if idx := strings.Index(name, "/"); idx != -1 {
			base = name[idx+1:]
		}
	}
	return strings.HasPrefix(base, "eslint-plugin") || strings.HasPrefix(base, "eslint-config")
}

// isAngularEslintPackage reports whether a package belongs to the angular-eslint
// ecosystem. "angular-eslint" is the meta-package; "@angular-eslint/*" are the
// individual plugin/parser/builder packages.
func isAngularEslintPackage(name string) bool {
	return name == "angular-eslint" || strings.HasPrefix(name, "@angular-eslint/")
}

// isTypescriptEslintPackage reports whether a package belongs to the typescript-eslint
// ecosystem. Internal utility packages (@typescript-eslint/utils, type-utils, etc.)
// are never directly referenced in config files, so the whole scope is skipped when
// any @typescript-eslint/ package is present.
func isTypescriptEslintPackage(name string) bool {
	return strings.HasPrefix(name, "@typescript-eslint/") || name == "typescript-eslint"
}

// isNxEcosystemPackage reports whether a package belongs to the Nx build system,
// including official (@nx/, @nrwl/) and community (@nxext/) plugins.
func isNxEcosystemPackage(name string) bool {
	return name == "nx" ||
		strings.HasPrefix(name, "@nx/") ||
		strings.HasPrefix(name, "@nrwl/") ||
		strings.HasPrefix(name, "@nxext/")
}

// isCapacitorEcosystemPackage reports whether a package belongs to the Capacitor
// mobile platform. Platform packages like @capacitor/ios and @capacitor/android
// are synced by the CLI and never imported directly in source files.
func isCapacitorEcosystemPackage(name string) bool {
	return strings.HasPrefix(name, "@capacitor/")
}

// isVitestEcosystemPackage reports whether a package is part of the vitest ecosystem.
//   - vitest itself: may be invoked only via CLI (globals mode) without imports
//   - vite: mandatory peer dep, not imported directly when using vitest/config
//   - @vitest/coverage-*: configured as provider: 'v8' string, never imported
//   - @vitest/ui: enabled via --ui CLI flag, never imported
//   - jsdom, happy-dom: test environments configured as strings, not imported
func isVitestEcosystemPackage(name string) bool {
	switch name {
	case "vitest", "vite", "jsdom", "happy-dom":
		return true
	}
	return strings.HasPrefix(name, "@vitest/")
}

// isAngularBuildPackage reports whether a package is part of the Angular build
// infrastructure. These packages are invoked by the Angular CLI / compiler and are
// referenced as builder strings in angular.json, not imported in application source.
func isAngularBuildPackage(name string) bool {
	if strings.HasPrefix(name, "@angular-devkit/") ||
		strings.HasPrefix(name, "@angular-eslint/") ||
		strings.HasPrefix(name, "@analogjs/") {
		return true
	}
	switch name {
	case "@angular/compiler-cli",
		"@angular/language-service",
		"@angular/build",
		"@angular/animations":
		return true
	}
	return false
}

// isCommitlintPackage reports whether a package belongs to the commitlint ecosystem.
func isCommitlintPackage(name string) bool {
	return strings.HasPrefix(name, "@commitlint/")
}

// isGraphqlCodegenPackage reports whether a package belongs to the GraphQL Code Generator
// ecosystem. Plugins and presets are referenced by shorthand name in codegen.yml
// (e.g. preset: 'near-operation-file') and never imported in application source.
func isGraphqlCodegenPackage(name string) bool {
	return strings.HasPrefix(name, "@graphql-codegen/")
}

// hasPrefixInDeps reports whether any installed dep name starts with prefix.
func hasPrefixInDeps(allDeps map[string]bool, prefix string) bool {
	for name := range allDeps {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// RunRemove runs `pnpm remove` for the given packages in dir.
func RunRemove(dir string, packages []string) (string, error) {
	if len(packages) == 0 {
		return "", nil
	}
	for _, name := range packages {
		if !validPackageName.MatchString(name) {
			return "", fmt.Errorf("refusing to remove: invalid package name %q", name)
		}
	}
	args := append([]string{"remove"}, packages...)
	cmd := exec.Command("pnpm", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("pnpm remove failed: %w", err)
	}
	return out.String(), nil
}
