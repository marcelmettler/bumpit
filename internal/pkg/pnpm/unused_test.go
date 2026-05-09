package pnpm

import (
	"testing"

	"github.com/marcelmettler/bumpit/internal/pkg"
)

// makeCtx builds a depCtx suitable for evalDep tests.
// allDeps defaults to an empty map; used defaults to an empty map.
func makeCtx(flags depCtx) depCtx {
	if flags.allDeps == nil {
		flags.allDeps = map[string]bool{}
	}
	if flags.used == nil {
		flags.used = map[string]bool{}
	}
	return flags
}

// ─── packageNameFromImport ───────────────────────────────────────────────────

func TestPackageNameFromImport(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"lodash", "lodash"},
		{"lodash/fp", "lodash"},
		{"@scope/pkg", "@scope/pkg"},
		{"@scope/pkg/sub/path", "@scope/pkg"},
		{"@scope/pkg/", "@scope/pkg"},
		{"react", "react"},
	}
	for _, c := range cases {
		if got := packageNameFromImport(c.in); got != c.want {
			t.Errorf("packageNameFromImport(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ─── atTypesBaseName ─────────────────────────────────────────────────────────

func TestAtTypesBaseName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"@types/react", "react"},
		{"@types/node", "node"},
		{"@types/babel__core", "@babel/core"},
		{"@types/babel__traverse", "@babel/traverse"},
		{"@types/lodash", "lodash"},
	}
	for _, c := range cases {
		if got := atTypesBaseName(c.in); got != c.want {
			t.Errorf("atTypesBaseName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ─── evalDep — TypeScript build toolchain ────────────────────────────────────

func TestEvalDep_TypescriptBuild(t *testing.T) {
	ctx := makeCtx(depCtx{isTS: true, allDeps: map[string]bool{"typescript": true}})
	for _, name := range []string{"typescript", "tslib", "ts-node", "ts-patch", "@swc/core", "@swc-node/register"} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with isTS=true: expected nil (skip), got %+v", name, got)
		}
	}
}

func TestEvalDep_TypescriptBuild_NoFlagSet(t *testing.T) {
	ctx := makeCtx(depCtx{isTS: false})
	// Without isTS flag, ts-node should NOT be auto-skipped.
	if got := evalDep("ts-node", pkg.DepDevDependencies, ctx); got == nil {
		t.Error("evalDep(ts-node) with isTS=false: expected unused, got nil")
	}
}

// ─── evalDep — @types/* ───────────────────────────────────────────────────────

func TestEvalDep_AtTypes_BaseUsed(t *testing.T) {
	ctx := makeCtx(depCtx{
		allDeps: map[string]bool{"react": true, "@types/react": true},
		used:    map[string]bool{"react": true},
	})
	if got := evalDep("@types/react", pkg.DepDevDependencies, ctx); got != nil {
		t.Errorf("@types/react with base used: expected nil, got %+v", got)
	}
}

func TestEvalDep_AtTypes_BaseNotUsed(t *testing.T) {
	ctx := makeCtx(depCtx{
		allDeps: map[string]bool{"react": true, "@types/react": true},
		used:    map[string]bool{},
	})
	if got := evalDep("@types/react", pkg.DepDevDependencies, ctx); got == nil {
		t.Error("@types/react with base unused: expected flagged, got nil")
	}
}

func TestEvalDep_AtTypes_NodeSkipped(t *testing.T) {
	// @types/node is always skipped — "node" is a runtime, not a dep.
	ctx := makeCtx(depCtx{})
	if got := evalDep("@types/node", pkg.DepDevDependencies, ctx); got != nil {
		t.Errorf("@types/node: expected nil (runtime skip), got %+v", got)
	}
}

func TestEvalDep_AtTypes_Scoped(t *testing.T) {
	// @types/babel__core → @babel/core; skip when @babel/core is used.
	ctx := makeCtx(depCtx{
		allDeps: map[string]bool{"@babel/core": true, "@types/babel__core": true},
		used:    map[string]bool{"@babel/core": true},
	})
	if got := evalDep("@types/babel__core", pkg.DepDevDependencies, ctx); got != nil {
		t.Errorf("@types/babel__core with @babel/core used: expected nil, got %+v", got)
	}
}

// ─── evalDep — ESLint ecosystem ───────────────────────────────────────────────

func TestEvalDep_EslintEcosystem(t *testing.T) {
	ctx := makeCtx(depCtx{hasEslint: true})
	packages := []string{
		"eslint",
		"eslint-plugin-import",
		"eslint-plugin-react",
		"eslint-config-prettier",
		"@prettier/eslint-plugin",
		"@typescript-eslint/eslint-plugin",
	}
	for _, name := range packages {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasEslint: expected nil, got %+v", name, got)
		}
	}
}

func TestEvalDep_EslintJs_NotAutoSkipped(t *testing.T) {
	// @eslint/js is NOT in the ESLint ecosystem predicate — it should only be
	// skipped if actually imported (caught by import scan → ctx.used).
	ctx := makeCtx(depCtx{hasEslint: true})
	if got := evalDep("@eslint/js", pkg.DepDevDependencies, ctx); got == nil {
		t.Error("@eslint/js: expected flagged when not imported, got nil")
	}
}

func TestEvalDep_EslintJs_UsedViaImport(t *testing.T) {
	ctx := makeCtx(depCtx{
		hasEslint: true,
		used:      map[string]bool{"@eslint/js": true},
	})
	if got := evalDep("@eslint/js", pkg.DepDevDependencies, ctx); got != nil {
		t.Errorf("@eslint/js imported: expected nil, got %+v", got)
	}
}

// ─── evalDep — angular-eslint ─────────────────────────────────────────────────

func TestEvalDep_AngularEslint_MetaPackage(t *testing.T) {
	// The "angular-eslint" meta-package (no @ prefix) must be skipped.
	ctx := makeCtx(depCtx{hasAngularEslint: true})
	if got := evalDep("angular-eslint", pkg.DepDevDependencies, ctx); got != nil {
		t.Errorf("angular-eslint meta-package: expected nil, got %+v", got)
	}
}

func TestEvalDep_AngularEslint_ScopedPackages(t *testing.T) {
	ctx := makeCtx(depCtx{hasAngularEslint: true})
	for _, name := range []string{
		"@angular-eslint/eslint-plugin",
		"@angular-eslint/eslint-plugin-template",
		"@angular-eslint/builder",
		"@angular-eslint/schematics",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasAngularEslint: expected nil, got %+v", name, got)
		}
	}
}

func TestEvalDep_AngularEslint_FlagOffWhenAbsent(t *testing.T) {
	// Without the flag, the package should be flagged as unused.
	ctx := makeCtx(depCtx{hasAngularEslint: false})
	if got := evalDep("angular-eslint", pkg.DepDevDependencies, ctx); got == nil {
		t.Error("angular-eslint without flag: expected flagged, got nil")
	}
}

// ─── evalDep — typescript-eslint ─────────────────────────────────────────────

func TestEvalDep_TypescriptEslint(t *testing.T) {
	ctx := makeCtx(depCtx{hasTypescriptEslint: true})
	for _, name := range []string{
		"typescript-eslint",
		"@typescript-eslint/parser",
		"@typescript-eslint/utils",
		"@typescript-eslint/type-utils",
		"@typescript-eslint/eslint-plugin",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasTypescriptEslint: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — Nx ────────────────────────────────────────────────────────────

func TestEvalDep_Nx(t *testing.T) {
	ctx := makeCtx(depCtx{hasNx: true})
	for _, name := range []string{"nx", "@nx/js", "@nx/angular", "@nrwl/workspace", "@nxext/stencil"} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasNx: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — Capacitor ─────────────────────────────────────────────────────

func TestEvalDep_Capacitor(t *testing.T) {
	ctx := makeCtx(depCtx{hasCapacitor: true})
	for _, name := range []string{
		"@capacitor/core",
		"@capacitor/ios",
		"@capacitor/android",
		"@capacitor/cli",
		"@capacitor/angular",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasCapacitor: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — Vitest ────────────────────────────────────────────────────────

func TestEvalDep_Vitest(t *testing.T) {
	ctx := makeCtx(depCtx{hasVitest: true})
	// vitest itself, vite (required peer), test environments, and all @vitest/* packages.
	for _, name := range []string{
		"vitest",
		"vite",
		"jsdom",
		"happy-dom",
		"@vitest/coverage-v8",
		"@vitest/coverage-istanbul",
		"@vitest/ui",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasVitest: expected nil, got %+v", name, got)
		}
	}
}

func TestEvalDep_Vitest_FlagOffWhenAbsent(t *testing.T) {
	ctx := makeCtx(depCtx{hasVitest: false})
	// vitest should be reported if the flag isn't set.
	if got := evalDep("vitest", pkg.DepDevDependencies, ctx); got == nil {
		t.Error("vitest without flag: expected flagged, got nil")
	}
}

// ─── evalDep — Angular build ─────────────────────────────────────────────────

func TestEvalDep_Angular(t *testing.T) {
	ctx := makeCtx(depCtx{hasAngular: true})
	for _, name := range []string{
		"@angular-devkit/build-angular",
		"@angular-devkit/core",
		"@analogjs/vite-plugin-angular",
		"@angular/compiler-cli",
		"@angular/language-service",
		"@angular/build",
		"@angular/animations",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasAngular: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — commitlint ────────────────────────────────────────────────────

func TestEvalDep_Commitlint(t *testing.T) {
	ctx := makeCtx(depCtx{hasCommitlint: true})
	for _, name := range []string{"@commitlint/cli", "@commitlint/config-conventional", "@commitlint/types"} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasCommitlint: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — GraphQL codegen ───────────────────────────────────────────────

func TestEvalDep_GraphqlCodegen(t *testing.T) {
	ctx := makeCtx(depCtx{hasGraphqlCodegen: true})
	for _, name := range []string{
		"@graphql-codegen/cli",
		"@graphql-codegen/typescript",
		"@graphql-codegen/typescript-operations",
		"@graphql-codegen/typescript-apollo-client-helpers",
		"@graphql-codegen/near-operation-file",
	} {
		if got := evalDep(name, pkg.DepDevDependencies, ctx); got != nil {
			t.Errorf("evalDep(%q) with hasGraphqlCodegen: expected nil, got %+v", name, got)
		}
	}
}

// ─── evalDep — used via import ───────────────────────────────────────────────

func TestEvalDep_UsedViaImport(t *testing.T) {
	ctx := makeCtx(depCtx{
		used: map[string]bool{"lodash": true, "@angular/core": true},
	})
	if got := evalDep("lodash", pkg.DepDependencies, ctx); got != nil {
		t.Errorf("lodash in ctx.used: expected nil, got %+v", got)
	}
	if got := evalDep("@angular/core", pkg.DepDependencies, ctx); got != nil {
		t.Errorf("@angular/core in ctx.used: expected nil, got %+v", got)
	}
}

func TestEvalDep_Unused(t *testing.T) {
	ctx := makeCtx(depCtx{})
	got := evalDep("some-random-package", pkg.DepDependencies, ctx)
	if got == nil {
		t.Fatal("some-random-package: expected flagged, got nil")
	}
	if got.Name != "some-random-package" {
		t.Errorf("got Name=%q, want %q", got.Name, "some-random-package")
	}
	if got.DepType != pkg.DepDependencies {
		t.Errorf("got DepType=%q, want %q", got.DepType, pkg.DepDependencies)
	}
}

// ─── collectScriptRefs — CLI aliases ─────────────────────────────────────────

func TestCollectScriptRefs_DirectName(t *testing.T) {
	scripts := map[string]string{"lint": "eslint src/"}
	allDeps := map[string]bool{"eslint": true}
	refs := collectScriptRefs(scripts, allDeps)
	if !refs["eslint"] {
		t.Error("eslint: expected in refs (direct name match)")
	}
}

func TestCollectScriptRefs_FirebaseTools(t *testing.T) {
	// "firebase-tools" binary is "firebase" — must be detected by CLI alias.
	scripts := map[string]string{"deploy": "firebase deploy --only hosting"}
	allDeps := map[string]bool{"firebase-tools": true}
	refs := collectScriptRefs(scripts, allDeps)
	if !refs["firebase-tools"] {
		t.Error("firebase-tools: expected in refs via alias 'firebase'")
	}
}

func TestCollectScriptRefs_GraphqlCodegenAlias(t *testing.T) {
	scripts := map[string]string{"codegen": "graphql-codegen --config codegen.yml"}
	allDeps := map[string]bool{"@graphql-codegen/cli": true}
	refs := collectScriptRefs(scripts, allDeps)
	if !refs["@graphql-codegen/cli"] {
		t.Error("@graphql-codegen/cli: expected in refs via alias 'graphql-codegen'")
	}
}

func TestCollectScriptRefs_NoMatch(t *testing.T) {
	scripts := map[string]string{"build": "tsc"}
	allDeps := map[string]bool{"webpack": true}
	refs := collectScriptRefs(scripts, allDeps)
	if refs["webpack"] {
		t.Error("webpack: expected NOT in refs (not in script)")
	}
}

// ─── addEslintConfigRefs ─────────────────────────────────────────────────────

func TestAddEslintConfigRefs_DoubleQuoted(t *testing.T) {
	content := []byte(`{ "extends": ["@typescript-eslint/recommended", "plugin:react/recommended"] }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if !refs["@typescript-eslint/recommended"] {
		t.Error("@typescript-eslint/recommended not in refs")
	}
	// "plugin:react/recommended" → eslint-plugin-react
	if !refs["eslint-plugin-react"] {
		t.Error("eslint-plugin-react (from plugin:react/recommended) not in refs")
	}
}

func TestAddEslintConfigRefs_SingleQuoted(t *testing.T) {
	content := []byte(`module.exports = { extends: ['@typescript-eslint/recommended'] }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if !refs["@typescript-eslint/recommended"] {
		t.Error("@typescript-eslint/recommended not in refs (single-quoted)")
	}
}

func TestAddEslintConfigRefs_PluginShorthand_Scoped(t *testing.T) {
	// "plugin:@typescript-eslint/recommended" → @typescript-eslint/eslint-plugin
	content := []byte(`{ "extends": "plugin:@typescript-eslint/recommended" }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if !refs["@typescript-eslint/eslint-plugin"] {
		t.Error("@typescript-eslint/eslint-plugin not in refs from plugin:@typescript-eslint/recommended")
	}
}

func TestAddEslintConfigRefs_PluginShorthand_Unscoped(t *testing.T) {
	// "plugin:react/recommended" → eslint-plugin-react
	content := []byte(`{ "extends": "plugin:react/recommended" }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if !refs["eslint-plugin-react"] {
		t.Error("eslint-plugin-react not in refs from plugin:react/recommended")
	}
}

func TestAddEslintConfigRefs_ScopeOnlyPlugin(t *testing.T) {
	// "@typescript-eslint" in a plugins array is shorthand for the eslint-plugin sub-package.
	content := []byte(`{ "plugins": ["@typescript-eslint"] }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	// Direct reference added as package name.
	if !refs["@typescript-eslint"] {
		t.Error("@typescript-eslint not in refs")
	}
	// Shorthand expansion.
	if !refs["@typescript-eslint/eslint-plugin"] {
		t.Error("@typescript-eslint/eslint-plugin not in refs from scope-only shorthand")
	}
}

func TestAddEslintConfigRefs_SkipsBuiltins(t *testing.T) {
	content := []byte(`{ "extends": ["eslint:recommended", "next:core-web-vitals", "./custom"] }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if refs["eslint:recommended"] || refs["next:core-web-vitals"] || refs["./custom"] {
		t.Error("built-in / relative refs should not be added")
	}
}

func TestAddEslintConfigRefs_AngularEslintRules(t *testing.T) {
	// Rules like '@angular-eslint/template/no-negated-async' should add @angular-eslint/template.
	content := []byte(`rules: { '@angular-eslint/template/no-negated-async': ['warn'] }`)
	refs := make(map[string]bool)
	addEslintConfigRefs(content, refs)
	if !refs["@angular-eslint/template"] {
		t.Error("@angular-eslint/template not in refs from rule key")
	}
}

// ─── isEslintConfigFile ───────────────────────────────────────────────────────

func TestIsEslintConfigFile(t *testing.T) {
	yes := []string{
		".eslintrc", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs",
		".eslintrc.yaml", ".eslintrc.yml",
		"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs",
		"eslint.config.ts", "eslint.config.mts",
	}
	for _, name := range yes {
		if !isEslintConfigFile(name) {
			t.Errorf("isEslintConfigFile(%q) = false, want true", name)
		}
	}
	no := []string{"package.json", "tsconfig.json", "nx.json", ".commitlintrc"}
	for _, name := range no {
		if isEslintConfigFile(name) {
			t.Errorf("isEslintConfigFile(%q) = true, want false", name)
		}
	}
}
