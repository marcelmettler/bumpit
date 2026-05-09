package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/marcelmettler/bumpit/internal/changelog"
	"github.com/marcelmettler/bumpit/internal/detect"
	"github.com/marcelmettler/bumpit/internal/pkg"
	"github.com/marcelmettler/bumpit/internal/pkg/gomod"
	"github.com/marcelmettler/bumpit/internal/pkg/pnpm"
	"github.com/marcelmettler/bumpit/internal/registry"
)

// appState tracks the current TUI state machine state.
type appState int

const (
	stateInit        appState = iota // detecting files
	stateLoading                     // fetching packages
	stateList                        // main interactive list
	stateDetail                      // changelog detail view
	stateUpdating                    // running pnpm update
	stateDone                        // update complete
	stateError                       // fatal error
	stateUnusedList                  // unused deps list
	stateRemoving                    // running removal
	stateRemoveDone                  // removal complete
)

// SortMode defines available sort orders.
type SortMode int

const (
	SortByKind SortMode = iota
	SortByName
	SortByAge
)

func (s SortMode) String() string {
	switch s {
	case SortByKind:
		return "update type (MAJOR first)"
	case SortByName:
		return "name"
	case SortByAge:
		return "age"
	default:
		return "unknown"
	}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ── Messages ──────────────────────────────────────────────────────────────────

type msgTick struct{}

type msgDetectDone struct {
	files []detect.PackageFile
}

type msgPackagesDone struct {
	packages []*pkg.PackageUpdate
	err      error
}

type msgRegistryDone struct {
	index int
	info  *registry.PackageInfo
	err   error
}

type msgChangelogDone struct {
	packageName string
	result      changelog.Result
}

type msgAuditDone struct {
	vulns map[string]int
}

type msgUpdateDone struct {
	output  string
	updated []*pkg.PackageUpdate
	err     error
}

type msgUnusedDone struct {
	packages []*pkg.UnusedPackage
	err      error
}

type msgRemoveDone struct {
	output  string
	removed []*pkg.UnusedPackage
	err     error
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Config holds startup options passed from the CLI.
type Config struct {
	ShowIndirect bool // include indirect Go module dependencies
	UnusedMode   bool // scan for unused deps instead of outdated
}

// Model is the bubbletea model for the entire application.
type Model struct {
	state  appState
	root   string
	width  int
	height int
	err    error

	// Config
	showIndirect bool
	unusedMode   bool

	// Spinner
	spinnerFrame int

	// Data
	files    []detect.PackageFile
	packages []*pkg.PackageUpdate

	// List view
	filtered     []*pkg.PackageUpdate
	cursor       int
	scrollOffset int
	filterQuery  string
	filterActive bool
	sortMode     SortMode
	showHelp     bool

	// Detail view
	detailPackage *pkg.PackageUpdate
	detailScroll  int

	// Update
	updatedPackages []*pkg.PackageUpdate
	updateOutput    string

	// Min release age
	minimumReleaseAge time.Duration

	// Status messages
	statusMsg string

	// Track changelog fetches in-flight
	changelogFetchedNames map[string]bool

	// Unused deps mode
	unusedPackages []*pkg.UnusedPackage
	unusedFiltered []*pkg.UnusedPackage
	unusedCursor   int
	unusedScroll   int

	// Remove result
	removedPackages []*pkg.UnusedPackage
	removeOutput    string
}

// New creates a new Model for the given root directory.
func New(root string, cfg Config) *Model {
	initDebug()
	return &Model{
		root:                  root,
		showIndirect:          cfg.ShowIndirect,
		unusedMode:            cfg.UnusedMode,
		state:                 stateInit,
		minimumReleaseAge:     3 * 24 * time.Hour,
		changelogFetchedNames: make(map[string]bool),
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return msgTick{} })
}

func isLoadingState(s appState) bool {
	return s == stateInit || s == stateLoading || s == stateUpdating || s == stateRemoving
}

// Init starts the detection phase.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.cmdDetect(), tickCmd())
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m *Model) cmdDetect() tea.Cmd {
	return func() tea.Msg {
		files, err := detect.Find(m.root)
		if err != nil {
			return msgPackagesDone{err: err}
		}
		return msgDetectDone{files: files}
	}
}

func (m *Model) cmdFetchPackages(files []detect.PackageFile) tea.Cmd {
	return func() tea.Msg {
		var all []*pkg.PackageUpdate

		for _, f := range files {
			switch {
			case f.PackageManager == detect.PackageManagerPNPM:
				if !pnpm.IsInstalled() {
					continue
				}
				pkgs, err := pnpm.Outdated(f.Dir, m.root)
				if err == nil {
					all = append(all, pkgs...)
				}
			case f.HasGoMod && gomod.IsInstalled():
				pkgs, err := gomod.Outdated(f.Dir, m.root)
				if err == nil {
					all = append(all, pkgs...)
				}
			}
		}

		return msgPackagesDone{packages: all}
	}
}

func (m *Model) cmdFetchRegistry(index int, p *pkg.PackageUpdate) tea.Cmd {
	return func() tea.Msg {
		if p.Source != "npm" {
			return msgRegistryDone{index: index} // no-op for go modules
		}
		info, err := registry.FetchPackageInfo(p.Name, p.Latest)
		return msgRegistryDone{index: index, info: info, err: err}
	}
}

func (m *Model) cmdFetchChangelog(p *pkg.PackageUpdate) tea.Cmd {
	return func() tea.Msg {
		result := changelog.Fetch(p)
		return msgChangelogDone{packageName: p.Name, result: result}
	}
}

func (m *Model) cmdRunAudit(dir string) tea.Cmd {
	return func() tea.Msg {
		vulns, _ := pnpm.AuditPackages(dir)
		return msgAuditDone{vulns: vulns}
	}
}

func (m *Model) cmdRunUpdate(packages []*pkg.PackageUpdate) tea.Cmd {
	return func() tea.Msg {
		// Group by directory
		byDir := make(map[string][]*pkg.PackageUpdate)
		for _, p := range packages {
			byDir[p.Dir] = append(byDir[p.Dir], p)
		}

		var allOutput strings.Builder
		for dir, pkgs := range byDir {
			var names []string
			for _, p := range pkgs {
				if p.Source == "npm" {
					names = append(names, p.Name)
				}
			}
			if len(names) == 0 {
				continue
			}
			out, err := pnpm.RunUpdate(dir, names)
			allOutput.WriteString(out)
			if err != nil {
				return msgUpdateDone{output: allOutput.String(), err: err, updated: packages}
			}
		}
		return msgUpdateDone{output: allOutput.String(), updated: packages}
	}
}

func (m *Model) cmdFindUnused(files []detect.PackageFile) tea.Cmd {
	return func() tea.Msg {
		// Walk the workspace once and share the result across all package.json files.
		ws := pnpm.ScanWorkspace(m.root)

		var all []*pkg.UnusedPackage
		for _, f := range files {
			switch {
			case f.PackageManager == detect.PackageManagerPNPM:
				if !pnpm.IsInstalled() {
					continue
				}
				pkgs, err := pnpm.FindUnused(f.Dir, m.root, ws)
				if err == nil {
					all = append(all, pkgs...)
				}
			case f.HasGoMod && gomod.IsInstalled():
				pkgs, err := gomod.FindUnused(f.Dir, m.root)
				if err == nil {
					all = append(all, pkgs...)
				}
			}
		}
		return msgUnusedDone{packages: all}
	}
}

func (m *Model) cmdRunRemove(packages []*pkg.UnusedPackage) tea.Cmd {
	return func() tea.Msg {
		type dirKey struct{ dir, source string }
		byDir := make(map[dirKey][]*pkg.UnusedPackage)
		for _, p := range packages {
			key := dirKey{p.Dir, p.Source}
			byDir[key] = append(byDir[key], p)
		}

		var allOutput strings.Builder
		for key, pkgs := range byDir {
			var names []string
			for _, p := range pkgs {
				names = append(names, p.Name)
			}
			var out string
			var err error
			if key.source == "npm" {
				out, err = pnpm.RunRemove(key.dir, names)
			} else {
				out, err = gomod.RunRemove(key.dir, names)
			}
			allOutput.WriteString(out)
			if err != nil {
				return msgRemoveDone{output: allOutput.String(), err: err, removed: packages}
			}
		}
		return msgRemoveDone{output: allOutput.String(), removed: packages}
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case msgDetectDone:
		m.files = msg.files
		m.state = stateLoading
		if m.unusedMode {
			m.statusMsg = fmt.Sprintf("Scanning %d package files for unused dependencies...", len(m.files))
			return m, m.cmdFindUnused(m.files)
		}
		m.statusMsg = fmt.Sprintf("Scanning %d package files...", len(m.files))
		return m, m.cmdFetchPackages(m.files)

	case msgPackagesDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.packages = msg.packages
		sortPackages(m.packages, m.sortMode)
		m.rebuildFiltered()
		m.state = stateList

		// Start fetching registry data for npm packages
		var cmds []tea.Cmd
		for i, p := range m.packages {
			if p.Source == "npm" {
				cmds = append(cmds, m.cmdFetchRegistry(i, p))
			}
		}

		// Run audit for pnpm dirs
		seen := make(map[string]bool)
		for _, f := range m.files {
			if f.PackageManager == detect.PackageManagerPNPM && !seen[f.Dir] {
				seen[f.Dir] = true
				cmds = append(cmds, m.cmdRunAudit(f.Dir))
			}
		}

		// Read minimum release age from first pnpm project
		for _, f := range m.files {
			if f.PackageManager == detect.PackageManagerPNPM {
				age := pnpm.NpmrcMinimumReleaseAge(f.Dir)
				m.minimumReleaseAge = registry.ParseMinimumReleaseAge(age)
				break
			}
		}

		return m, tea.Batch(cmds...)

	case msgRegistryDone:
		if msg.info != nil && msg.index < len(m.packages) {
			p := m.packages[msg.index]
			p.PublishedAt = msg.info.PublishedAt
			p.RepositoryURL = msg.info.RepositoryURL

			// Evaluate eligibility
			if !p.PublishedAt.IsZero() {
				eligibleAt := p.PublishedAt.Add(m.minimumReleaseAge)
				p.EligibleAt = eligibleAt
				p.IsEligible = time.Now().After(eligibleAt)
			} else {
				p.IsEligible = true
			}

			// Proactively fetch changelog for MAJOR updates first, then others
			if !m.changelogFetchedNames[p.Name] {
				m.changelogFetchedNames[p.Name] = true
				return m, m.cmdFetchChangelog(p)
			}
		}
		return m, nil

	case msgChangelogDone:
		for _, p := range m.packages {
			if p.Name == msg.packageName {
				p.Changelog = msg.result.Markdown
				p.HasBreaking = msg.result.HasBreaking
				p.OneLineSummary = msg.result.Summary
				p.Highlights = msg.result.Highlights
				p.ChangelogFetched = true
				break
			}
		}
		return m, nil

	case msgAuditDone:
		for _, p := range m.packages {
			if count, ok := msg.vulns[p.Name]; ok {
				p.VulnCount = count
			}
		}
		return m, nil

	case msgUpdateDone:
		m.updatedPackages = msg.updated
		m.updateOutput = msg.output
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateDone
		}
		return m, nil

	case msgUnusedDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.unusedPackages = msg.packages
		sortUnused(m.unusedPackages)
		m.rebuildUnusedFiltered()
		m.state = stateUnusedList
		return m, nil

	case msgRemoveDone:
		m.removedPackages = msg.removed
		m.removeOutput = msg.output
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateRemoveDone
		}
		return m, nil

	case msgTick:
		m.spinnerFrame++
		if isLoadingState(m.state) {
			return m, tickCmd()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dbg("handleKey: key=%q state=%d", msg.String(), m.state)
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	switch m.state {
	case stateList:
		return m.updateList(msg)
	case stateDetail:
		return m.updateDetail(msg)
	case stateDone:
		return m, tea.Quit
	case stateUnusedList:
		return m.updateUnusedList(msg)
	case stateRemoveDone:
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updateUnusedList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.unusedCursor < len(m.unusedFiltered)-1 {
			m.unusedCursor++
		}

	case "k", "up":
		if m.unusedCursor > 0 {
			m.unusedCursor--
		}

	case " ":
		if len(m.unusedFiltered) > 0 {
			m.unusedFiltered[m.unusedCursor].Selected = !m.unusedFiltered[m.unusedCursor].Selected
		}

	case "a":
		allSelected := true
		for _, p := range m.unusedFiltered {
			if !p.Selected {
				allSelected = false
				break
			}
		}
		for _, p := range m.unusedFiltered {
			p.Selected = !allSelected
		}

	case "r":
		selected := selectedUnused(m.unusedPackages)
		if len(selected) == 0 {
			m.statusMsg = "No packages selected. Press SPACE to select."
		} else {
			m.state = stateRemoving
			return m, m.cmdRunRemove(selected)
		}
	}
	return m, nil
}

func (m *Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Filter input mode
	if m.filterActive {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.filterActive = false
			m.filterQuery = ""
			m.rebuildFiltered()
		case "enter":
			m.filterActive = false
			m.rebuildFiltered()
		case "backspace":
			if len(m.filterQuery) > 0 {
				m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
				m.rebuildFiltered()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.filterQuery += string(msg.Runes)
				m.rebuildFiltered()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp

	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case " ":
		if len(m.filtered) > 0 {
			m.filtered[m.cursor].Selected = !m.filtered[m.cursor].Selected
		}

	case "a":
		// Select/deselect all visible
		allSelected := true
		for _, p := range m.filtered {
			if !p.Selected {
				allSelected = false
				break
			}
		}
		for _, p := range m.filtered {
			p.Selected = !allSelected
		}

	case "enter":
		if len(m.filtered) > 0 {
			m.detailPackage = m.filtered[m.cursor]
			m.detailScroll = 0
			m.state = stateDetail
		}

	case "/":
		m.filterActive = true

	case "s":
		m.sortMode = (m.sortMode + 1) % 3
		sortPackages(m.packages, m.sortMode)
		m.rebuildFiltered()

	case "u":
		selected := selectedPackages(m.packages)
		if len(selected) == 0 {
			m.statusMsg = "No packages selected. Press SPACE to select."
		} else {
			m.state = stateUpdating
			return m, m.cmdRunUpdate(selected)
		}
	}

	return m, nil
}

func (m *Model) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	dbg("updateDetail: key=%q scroll=%d", msg.String(), m.detailScroll)
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = stateList
		m.detailPackage = nil

	case "j", "down":
		m.detailScroll++

	case "k", "up":
		if m.detailScroll > 0 {
			m.detailScroll--
		}

	case " ":
		if m.detailPackage != nil {
			m.detailPackage.Selected = !m.detailPackage.Selected
		}

	case "u":
		if m.detailPackage != nil {
			m.state = stateUpdating
			return m, m.cmdRunUpdate([]*pkg.PackageUpdate{m.detailPackage})
		}
	}

	return m, nil
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m *Model) View() string {
	spin := spinnerFrames[m.spinnerFrame%len(spinnerFrames)]

	switch m.state {
	case stateInit:
		return renderLoadingScreen(m, spin+"  Detecting package files...")

	case stateLoading:
		return renderLoadingScreen(m, spin+"  "+m.statusMsg)

	case stateList:
		if m.showHelp {
			return renderHelpOverlay()
		}
		return renderList(m)

	case stateDetail:
		return renderDetail(m)

	case stateDone:
		return renderUpdateSummary(m.updatedPackages, m.updateOutput)

	case stateUpdating:
		return renderLoadingScreen(m, spin+"  Running updates...")

	case stateUnusedList:
		return renderUnusedList(m)

	case stateRemoving:
		return renderLoadingScreen(m, spin+"  Removing packages...")

	case stateRemoveDone:
		return renderRemoveSummary(m.removedPackages, m.removeOutput)

	case stateError:
		errMsg := "unknown error"
		if m.err != nil {
			errMsg = m.err.Error()
		}
		return styleBreakingBanner.Render("\n  Error: "+errMsg) + "\n\n" +
			styleMuted.Render("  Press q to quit.") + "\n"
	}

	return ""
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (m *Model) rebuildFiltered() {
	q := strings.ToLower(m.filterQuery)
	var filtered []*pkg.PackageUpdate
	for _, p := range m.packages {
		if !m.showIndirect && p.DepType == pkg.DepIndirect {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.DirName), q) {
			filtered = append(filtered, p)
		}
	}
	m.filtered = filtered

	// Clamp cursor
	if m.cursor >= len(m.filtered) {
		m.cursor = clampMin(0, len(m.filtered)-1)
	}
}

func sortPackages(packages []*pkg.PackageUpdate, mode SortMode) {
	sort.SliceStable(packages, func(i, j int) bool {
		a, b := packages[i], packages[j]
		switch mode {
		case SortByName:
			return a.Name < b.Name
		case SortByAge:
			if a.PublishedAt.IsZero() && b.PublishedAt.IsZero() {
				return a.Name < b.Name
			}
			if a.PublishedAt.IsZero() {
				return false
			}
			if b.PublishedAt.IsZero() {
				return true
			}
			return a.PublishedAt.After(b.PublishedAt) // newest first
		default: // SortByKind
			if a.Kind != b.Kind {
				return a.Kind > b.Kind // MAJOR > minor > patch
			}
			return a.Name < b.Name
		}
	})
}

func selectedPackages(packages []*pkg.PackageUpdate) []*pkg.PackageUpdate {
	var sel []*pkg.PackageUpdate
	for _, p := range packages {
		if p.Selected {
			sel = append(sel, p)
		}
	}
	return sel
}

func clampMin(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func selectedUnused(packages []*pkg.UnusedPackage) []*pkg.UnusedPackage {
	var sel []*pkg.UnusedPackage
	for _, p := range packages {
		if p.Selected {
			sel = append(sel, p)
		}
	}
	return sel
}

func sortUnused(packages []*pkg.UnusedPackage) {
	sort.SliceStable(packages, func(i, j int) bool {
		a, b := packages[i], packages[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Name < b.Name
	})
}

func (m *Model) rebuildUnusedFiltered() {
	m.unusedFiltered = make([]*pkg.UnusedPackage, len(m.unusedPackages))
	copy(m.unusedFiltered, m.unusedPackages)
	if m.unusedCursor >= len(m.unusedFiltered) {
		m.unusedCursor = clampMin(0, len(m.unusedFiltered)-1)
	}
}
