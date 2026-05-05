package detect

import (
	"os"
	"path/filepath"
	"strings"
)

// PackageManager represents the type of package manager detected.
type PackageManager int

const (
	PackageManagerNone PackageManager = iota
	PackageManagerPNPM
	PackageManagerYarn
	PackageManagerNPM
)

// PackageFile represents a discovered package file.
type PackageFile struct {
	Dir            string
	PackageManager PackageManager
	HasGoMod       bool
}

var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
	"vendor":       true,
}

// Find walks root recursively and returns discovered package files.
func Find(root string) ([]PackageFile, error) {
	var results []PackageFile

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		dir := filepath.Dir(path)
		name := d.Name()

		switch name {
		case "package.json":
			pm := detectPackageManager(dir)
			results = appendOrUpdate(results, PackageFile{
				Dir:            dir,
				PackageManager: pm,
			})
		case "go.mod":
			results = appendOrUpdate(results, PackageFile{
				Dir:      dir,
				HasGoMod: true,
			})
		}

		return nil
	})

	return results, err
}

// detectPackageManager determines which package manager is used in the directory.
func detectPackageManager(dir string) PackageManager {
	// Check lockfiles to determine package manager
	if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
		return PackageManagerPNPM
	}
	if fileExists(filepath.Join(dir, "yarn.lock")) {
		return PackageManagerYarn
	}
	if fileExists(filepath.Join(dir, "package-lock.json")) {
		return PackageManagerNPM
	}

	// Check parent dirs for pnpm workspace
	parent := filepath.Dir(dir)
	for parent != dir {
		if fileExists(filepath.Join(parent, "pnpm-lock.yaml")) {
			return PackageManagerPNPM
		}
		if fileExists(filepath.Join(parent, "pnpm-workspace.yaml")) {
			return PackageManagerPNPM
		}
		dir = parent
		parent = filepath.Dir(dir)
	}

	return PackageManagerNPM
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ShortName returns a display-friendly short name for the directory.
func ShortName(dir, root string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return filepath.Base(dir)
	}
	if rel == "." {
		return "root"
	}
	// Trim long paths
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return rel
}

// appendOrUpdate adds a PackageFile to results, merging if same dir exists.
func appendOrUpdate(results []PackageFile, pf PackageFile) []PackageFile {
	for i, r := range results {
		if r.Dir == pf.Dir {
			if pf.PackageManager != PackageManagerNone {
				results[i].PackageManager = pf.PackageManager
			}
			if pf.HasGoMod {
				results[i].HasGoMod = true
			}
			return results
		}
	}
	return append(results, pf)
}
