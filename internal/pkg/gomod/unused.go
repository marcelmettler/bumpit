package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/marcelmettler/chorekit/internal/detect"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

// validModulePath matches standard Go module path characters.
var validModulePath = regexp.MustCompile(`^[a-zA-Z0-9./\-_]+$`)

type goModuleEntry struct {
	Path     string `json:"Path"`
	Indirect bool   `json:"Indirect"`
	Main     bool   `json:"Main"`
}

type goPackageEntry struct {
	Imports      []string `json:"Imports"`
	TestImports  []string `json:"TestImports"`
	XTestImports []string `json:"XTestImports"`
}

// FindUnused returns direct Go module dependencies (go.mod require entries without // indirect)
// that are not imported anywhere in the project, including test files.
func FindUnused(dir, root string) ([]*pkg.UnusedPackage, error) {
	directModules, err := listDirectGoModules(dir)
	if err != nil {
		return nil, err
	}
	if len(directModules) == 0 {
		return nil, nil
	}

	imports, err := listGoProjectImports(dir)
	if err != nil {
		return nil, err
	}

	dirName := detect.ShortName(dir, root)
	var unused []*pkg.UnusedPackage
	for _, mod := range directModules {
		if !isGoModuleImported(mod.Path, imports) {
			unused = append(unused, &pkg.UnusedPackage{
				Name:    mod.Path,
				Dir:     dir,
				DirName: dirName,
				Source:  "go",
				DepType: pkg.DepDependencies,
			})
		}
	}
	return unused, nil
}

func listDirectGoModules(dir string) ([]goModuleEntry, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list -m failed: %w\n%s", err, stderr.String())
	}

	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var modules []goModuleEntry
	for dec.More() {
		var m goModuleEntry
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		if !m.Main && !m.Indirect {
			modules = append(modules, m)
		}
	}
	return modules, nil
}

func listGoProjectImports(dir string) (map[string]bool, error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list ./... failed: %w\n%s", err, stderr.String())
	}

	imports := make(map[string]bool)
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	for dec.More() {
		var p goPackageEntry
		if err := dec.Decode(&p); err != nil {
			return nil, err
		}
		for _, imp := range p.Imports {
			imports[imp] = true
		}
		for _, imp := range p.TestImports {
			imports[imp] = true
		}
		for _, imp := range p.XTestImports {
			imports[imp] = true
		}
	}
	return imports, nil
}

func isGoModuleImported(modulePath string, imports map[string]bool) bool {
	for imp := range imports {
		if imp == modulePath || strings.HasPrefix(imp, modulePath+"/") {
			return true
		}
	}
	return false
}

// RunRemove removes Go modules by running `go get mod@none` for each, then `go mod tidy`.
func RunRemove(dir string, modules []string) (string, error) {
	if len(modules) == 0 {
		return "", nil
	}
	var out strings.Builder
	for _, mod := range modules {
		if !validModulePath.MatchString(mod) {
			return out.String(), fmt.Errorf("refusing to remove: invalid module path %q", mod)
		}
		cmd := exec.Command("go", "get", mod+"@none")
		cmd.Dir = dir
		var b bytes.Buffer
		cmd.Stdout = &b
		cmd.Stderr = &b
		_ = cmd.Run() // errors expected when module is already absent
		out.WriteString(b.String())
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	if err := cmd.Run(); err != nil {
		return out.String() + b.String(), fmt.Errorf("go mod tidy failed: %w", err)
	}
	out.WriteString(b.String())
	return out.String(), nil
}
