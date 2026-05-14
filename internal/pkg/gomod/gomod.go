package gomod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Masterminds/semver/v3"
	"github.com/marcelmettler/chorekit/internal/detect"
	"github.com/marcelmettler/chorekit/internal/pkg"
)

// listModule represents one entry from `go list -m -u -json all`.
type listModule struct {
	Path     string      `json:"Path"`
	Version  string      `json:"Version"`
	Indirect bool        `json:"Indirect"`
	Update   *listUpdate `json:"Update"`
}

type listUpdate struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
}

// IsInstalled reports whether the go tool is available.
func IsInstalled() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// Outdated runs `go list -m -u -json all` in dir and returns packages with updates.
func Outdated(dir, root string) ([]*pkg.PackageUpdate, error) {
	cmd := exec.Command("go", "list", "-m", "-u", "-json", "all")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w\n%s", err, stderr.String())
	}

	modules, err := parseStreamingJSON(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse go list output: %w", err)
	}

	dirName := detect.ShortName(dir, root)
	var updates []*pkg.PackageUpdate
	for _, m := range modules {
		if m.Update == nil {
			continue
		}
		kind := classifyUpdate(m.Version, m.Update.Version)
		depType := pkg.DepDependencies
		if m.Indirect {
			depType = pkg.DepIndirect
		}
		updates = append(updates, &pkg.PackageUpdate{
			Name:       m.Path,
			Dir:        dir,
			DirName:    dirName,
			Source:     "go",
			Current:    m.Version,
			Wanted:     m.Update.Version,
			Latest:     m.Update.Version,
			Kind:       kind,
			DepType:    depType,
			IsEligible: true,
		})
	}

	return updates, nil
}

// parseStreamingJSON parses a stream of concatenated JSON objects (go list output).
func parseStreamingJSON(data []byte) ([]listModule, error) {
	var modules []listModule
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var m listModule
		if err := dec.Decode(&m); err != nil {
			return modules, err
		}
		modules = append(modules, m)
	}
	return modules, nil
}

func classifyUpdate(current, latest string) pkg.UpdateKind {
	// Go versions look like v0.20.0
	cur, err1 := semver.NewVersion(current)
	lat, err2 := semver.NewVersion(latest)
	if err1 != nil || err2 != nil {
		return pkg.KindUnknown
	}
	switch {
	case lat.Major() > cur.Major():
		return pkg.KindMajor
	case lat.Minor() > cur.Minor():
		return pkg.KindMinor
	case lat.Patch() > cur.Patch():
		return pkg.KindPatch
	default:
		return pkg.KindPatch
	}
}
