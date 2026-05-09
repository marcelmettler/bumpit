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
	cleanpkg "github.com/marcelmettler/bumpit/internal/pkg/clean"
	csspkg  "github.com/marcelmettler/bumpit/internal/pkg/css"
	i18npkg "github.com/marcelmettler/bumpit/internal/pkg/i18n"
	todopkg "github.com/marcelmettler/bumpit/internal/pkg/todo"
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
	stateLicenseList                 // license audit list
	stateCleanList                   // clean workspace list
	stateCleanDeleting               // running deletion
	stateCleanDone                   // deletion complete
	stateCSSList                     // CSS unused-class audit
	stateTodoList                    // TODO/FIXME/HACK/XXX audit
	stateI18nList                    // i18n key audit
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

type msgLicenseDone struct {
	packages []*pkg.LicenseInfo
	err      error
}

type msgCleanScanDone struct {
	artifacts []*pkg.ArtifactDir
	err       error
}

type msgCleanDeleteDone struct {
	freed   int64
	removed []*pkg.ArtifactDir
	err     error
}

type msgCSSScanDone struct {
	result *csspkg.ScanResult
	err    error
}

type msgTodoScanDone struct {
	result *todopkg.ScanResult
	err    error
}

type msgI18nScanDone struct {
	result *i18npkg.ScanResult
	err    error
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Config holds startup options passed from the CLI.
type Config struct {
	ShowIndirect bool // include indirect Go module dependencies
	UnusedMode   bool // scan for unused deps instead of outdated
	LicenseMode  bool // audit dependency licenses
	CleanMode    bool // find and delete artifact directories
	CSSMode      bool // audit unused CSS classes
	TodoMode     bool // scan for TODO/FIXME/HACK/XXX comments
	I18nMode     bool // audit i18n translation keys
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
	licenseMode  bool
	cleanMode    bool
	cssMode      bool
	todoMode     bool
	i18nMode     bool

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

	// Clean mode
	cleanArtifacts []*pkg.ArtifactDir
	cleanCursor    int
	cleanScroll    int
	cleanFreed     int64
	cleanRemoved   []*pkg.ArtifactDir

	// License mode
	licensePackages     []*pkg.LicenseInfo
	licenseFiltered     []*pkg.LicenseInfo
	licenseCursor       int
	licenseScroll       int
	licenseFilterQuery  string
	licenseFilterActive bool
	licenseSortByName   bool
	licenseShowAll      bool

	// CSS mode
	cssResult       *csspkg.ScanResult
	cssItems        []cssItem // combined filtered list: unused + undefined
	cssCursor       int
	cssScroll       int
	cssFilterQuery  string
	cssFilterActive bool

	// Todo mode
	todoResult       *todopkg.ScanResult
	todoFiltered     []*pkg.TodoItem
	todoCursor       int
	todoScroll       int
	todoFilterQuery  string
	todoFilterActive bool

	// i18n mode
	i18nResult       *i18npkg.ScanResult
	i18nItems        []i18nItem
	i18nCursor       int
	i18nScroll       int
	i18nFilterQuery  string
	i18nFilterActive bool
}

// New creates a new Model for the given root directory.
func New(root string, cfg Config) *Model {
	initDebug()
	return &Model{
		root:                  root,
		showIndirect:          cfg.ShowIndirect,
		unusedMode:            cfg.UnusedMode,
		licenseMode:           cfg.LicenseMode,
		cleanMode:             cfg.CleanMode,
		cssMode:               cfg.CSSMode,
		todoMode:              cfg.TodoMode,
		i18nMode:              cfg.I18nMode,
		state:                 stateInit,
		minimumReleaseAge:     3 * 24 * time.Hour,
		changelogFetchedNames: make(map[string]bool),
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return msgTick{} })
}

func isLoadingState(s appState) bool {
	return s == stateInit || s == stateLoading || s == stateUpdating || s == stateRemoving || s == stateCleanDeleting
}

// Init starts the detection phase.
func (m *Model) Init() tea.Cmd {
	if m.cssMode {
		m.state = stateLoading
		m.statusMsg = "Scanning CSS files..."
		return tea.Batch(m.cmdScanCSS(), tickCmd())
	}
	if m.todoMode {
		m.state = stateLoading
		m.statusMsg = "Scanning for TODO comments..."
		return tea.Batch(m.cmdScanTodo(), tickCmd())
	}
	if m.i18nMode {
		m.state = stateLoading
		m.statusMsg = "Scanning locale files and source references..."
		return tea.Batch(m.cmdScanI18n(), tickCmd())
	}
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
		if m.licenseMode {
			m.statusMsg = fmt.Sprintf("Reading licenses for %d package files...", len(m.files))
			return m, m.cmdFindLicenses(m.files)
		}
		if m.cleanMode {
			m.statusMsg = "Scanning for artifact directories..."
			return m, m.cmdScanArtifacts()
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

	case msgLicenseDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.licensePackages = msg.packages
		m.rebuildLicenseFiltered()
		m.state = stateLicenseList
		return m, nil

	case msgCleanScanDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.cleanArtifacts = msg.artifacts
		m.state = stateCleanList
		return m, nil

	case msgCleanDeleteDone:
		m.cleanFreed = msg.freed
		m.cleanRemoved = msg.removed
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateCleanDone
		}
		return m, nil

	case msgCSSScanDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.cssResult = msg.result
		m.rebuildCSSFiltered()
		m.state = stateCSSList
		return m, nil

	case msgTodoScanDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.todoResult = msg.result
		m.rebuildTodoFiltered()
		m.state = stateTodoList
		return m, nil

	case msgI18nScanDone:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
			return m, nil
		}
		m.i18nResult = msg.result
		m.rebuildI18nFiltered()
		m.state = stateI18nList
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
	case stateLicenseList:
		return m.updateLicenseList(msg)
	case stateCleanList:
		return m.updateCleanList(msg)
	case stateCleanDone:
		return m, tea.Quit
	case stateCSSList:
		return m.updateCSSList(msg)
	case stateTodoList:
		return m.updateTodoList(msg)
	case stateI18nList:
		return m.updateI18nList(msg)
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

	case stateLicenseList:
		return renderLicenseList(m)

	case stateCleanList:
		return renderCleanList(m)

	case stateCleanDeleting:
		return renderLoadingScreen(m, spin+"  Deleting selected directories...")

	case stateCleanDone:
		return renderCleanDone(m.cleanFreed, m.cleanRemoved)

	case stateCSSList:
		return renderCSSList(m)

	case stateTodoList:
		return renderTodoList(m)

	case stateI18nList:
		return renderI18nList(m)

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

func (m *Model) cmdScanArtifacts() tea.Cmd {
	return func() tea.Msg {
		artifacts, err := cleanpkg.FindArtifacts(m.root)
		return msgCleanScanDone{artifacts: artifacts, err: err}
	}
}

func (m *Model) cmdDeleteArtifacts(artifacts []*pkg.ArtifactDir) tea.Cmd {
	return func() tea.Msg {
		freed, err := cleanpkg.Remove(artifacts)
		return msgCleanDeleteDone{freed: freed, removed: artifacts, err: err}
	}
}

func (m *Model) updateCleanList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cleanCursor < len(m.cleanArtifacts)-1 {
			m.cleanCursor++
		}
	case "k", "up":
		if m.cleanCursor > 0 {
			m.cleanCursor--
		}
	case " ":
		if len(m.cleanArtifacts) > 0 {
			m.cleanArtifacts[m.cleanCursor].Selected = !m.cleanArtifacts[m.cleanCursor].Selected
		}
	case "a":
		allSelected := true
		for _, a := range m.cleanArtifacts {
			if !a.Selected {
				allSelected = false
				break
			}
		}
		for _, a := range m.cleanArtifacts {
			a.Selected = !allSelected
		}
	case "D":
		sel := selectedArtifacts(m.cleanArtifacts)
		if len(sel) == 0 {
			m.statusMsg = "No directories selected. Press SPACE to select."
		} else {
			m.state = stateCleanDeleting
			return m, m.cmdDeleteArtifacts(sel)
		}
	}
	return m, nil
}

func (m *Model) cmdFindLicenses(files []detect.PackageFile) tea.Cmd {
	return func() tea.Msg {
		seen := make(map[string]bool)
		var all []*pkg.LicenseInfo
		for _, f := range files {
			if f.PackageManager != detect.PackageManagerPNPM {
				continue
			}
			pkgs, err := pnpm.FindLicenses(f.Dir, m.root)
			if err != nil {
				continue
			}
			for _, p := range pkgs {
				if !seen[p.Name] {
					seen[p.Name] = true
					all = append(all, p)
				}
			}
		}
		sortLicenses(all, false)
		return msgLicenseDone{packages: all}
	}
}

func (m *Model) updateLicenseList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""

	if m.licenseFilterActive {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.licenseFilterActive = false
			m.licenseFilterQuery = ""
			m.rebuildLicenseFiltered()
		case "enter":
			m.licenseFilterActive = false
		case "backspace":
			if len(m.licenseFilterQuery) > 0 {
				m.licenseFilterQuery = m.licenseFilterQuery[:len(m.licenseFilterQuery)-1]
				m.rebuildLicenseFiltered()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.licenseFilterQuery += string(msg.Runes)
				m.rebuildLicenseFiltered()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.licenseCursor < len(m.licenseFiltered)-1 {
			m.licenseCursor++
		}
	case "k", "up":
		if m.licenseCursor > 0 {
			m.licenseCursor--
		}
	case "a":
		m.licenseShowAll = !m.licenseShowAll
		m.licenseCursor = 0
		m.licenseScroll = 0
		m.licenseFilterQuery = ""
		m.rebuildLicenseFiltered()
	case "/":
		if m.licenseShowAll {
			m.licenseFilterActive = true
		}
	case "s":
		if m.licenseShowAll {
			m.licenseSortByName = !m.licenseSortByName
			sortLicenses(m.licensePackages, m.licenseSortByName)
			m.rebuildLicenseFiltered()
		}
	}
	return m, nil
}

func (m *Model) rebuildLicenseFiltered() {
	q := strings.ToLower(m.licenseFilterQuery)
	var filtered []*pkg.LicenseInfo
	for _, p := range m.licensePackages {
		// In summary mode show only packages that need attention.
		if !m.licenseShowAll && p.Category == pkg.LicenseCategoryPermissive {
			continue
		}
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(strings.ToLower(p.License), q) {
			filtered = append(filtered, p)
		}
	}
	m.licenseFiltered = filtered
	if m.licenseCursor >= len(m.licenseFiltered) {
		m.licenseCursor = clampMin(0, len(m.licenseFiltered)-1)
	}
}

func sortLicenses(packages []*pkg.LicenseInfo, byName bool) {
	sort.SliceStable(packages, func(i, j int) bool {
		a, b := packages[i], packages[j]
		if byName {
			return a.Name < b.Name
		}
		if a.Category != b.Category {
			return a.Category < b.Category // risky (low int) first
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

func (m *Model) cmdScanCSS() tea.Cmd {
	return func() tea.Msg {
		result, err := csspkg.Scan(m.root)
		return msgCSSScanDone{result: result, err: err}
	}
}

func (m *Model) updateCSSList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""

	if m.cssFilterActive {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.cssFilterActive = false
			m.cssFilterQuery = ""
			m.rebuildCSSFiltered()
		case "enter":
			m.cssFilterActive = false
		case "backspace":
			if len(m.cssFilterQuery) > 0 {
				m.cssFilterQuery = m.cssFilterQuery[:len(m.cssFilterQuery)-1]
				m.rebuildCSSFiltered()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.cssFilterQuery += string(msg.Runes)
				m.rebuildCSSFiltered()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.cssCursor < len(m.cssItems)-1 {
			m.cssCursor++
		}
	case "k", "up":
		if m.cssCursor > 0 {
			m.cssCursor--
		}
	case "/":
		m.cssFilterActive = true
	}
	return m, nil
}

func (m *Model) cmdScanTodo() tea.Cmd {
	return func() tea.Msg {
		result, err := todopkg.Scan(m.root)
		return msgTodoScanDone{result: result, err: err}
	}
}

func (m *Model) updateTodoList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""

	if m.todoFilterActive {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.todoFilterActive = false
			m.todoFilterQuery = ""
			m.rebuildTodoFiltered()
		case "enter":
			m.todoFilterActive = false
		case "backspace":
			if len(m.todoFilterQuery) > 0 {
				m.todoFilterQuery = m.todoFilterQuery[:len(m.todoFilterQuery)-1]
				m.rebuildTodoFiltered()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.todoFilterQuery += string(msg.Runes)
				m.rebuildTodoFiltered()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.todoCursor < len(m.todoFiltered)-1 {
			m.todoCursor++
		}
	case "k", "up":
		if m.todoCursor > 0 {
			m.todoCursor--
		}
	case "/":
		m.todoFilterActive = true
	}
	return m, nil
}

func (m *Model) rebuildTodoFiltered() {
	if m.todoResult == nil {
		m.todoFiltered = nil
		return
	}
	q := strings.ToLower(m.todoFilterQuery)
	var filtered []*pkg.TodoItem
	for _, item := range m.todoResult.Items {
		if q == "" ||
			strings.Contains(strings.ToLower(item.Kind), q) ||
			strings.Contains(strings.ToLower(item.Text), q) ||
			strings.Contains(strings.ToLower(item.File), q) {
			filtered = append(filtered, item)
		}
	}
	m.todoFiltered = filtered
	if m.todoCursor >= len(m.todoFiltered) {
		m.todoCursor = clampMin(0, len(m.todoFiltered)-1)
	}
}

func (m *Model) cmdScanI18n() tea.Cmd {
	return func() tea.Msg {
		result, err := i18npkg.Scan(m.root)
		return msgI18nScanDone{result: result, err: err}
	}
}

func (m *Model) updateI18nList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""

	if m.i18nFilterActive {
		switch msg.String() {
		case "esc", "ctrl+c":
			m.i18nFilterActive = false
			m.i18nFilterQuery = ""
			m.rebuildI18nFiltered()
		case "enter":
			m.i18nFilterActive = false
		case "backspace":
			if len(m.i18nFilterQuery) > 0 {
				m.i18nFilterQuery = m.i18nFilterQuery[:len(m.i18nFilterQuery)-1]
				m.rebuildI18nFiltered()
			}
		default:
			if msg.Type == tea.KeyRunes {
				m.i18nFilterQuery += string(msg.Runes)
				m.rebuildI18nFiltered()
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.i18nCursor < len(m.i18nItems)-1 {
			m.i18nCursor++
		}
	case "k", "up":
		if m.i18nCursor > 0 {
			m.i18nCursor--
		}
	case "/":
		m.i18nFilterActive = true
	}
	return m, nil
}

func (m *Model) rebuildI18nFiltered() {
	if m.i18nResult == nil {
		m.i18nItems = nil
		return
	}
	q := strings.ToLower(m.i18nFilterQuery)
	var items []i18nItem
	for _, k := range m.i18nResult.Unused {
		if q == "" || strings.Contains(strings.ToLower(k.Key), q) ||
			strings.Contains(strings.ToLower(k.File), q) {
			items = append(items, i18nItem{key: k, undefined: false})
		}
	}
	for _, k := range m.i18nResult.Undefined {
		if q == "" || strings.Contains(strings.ToLower(k.Key), q) ||
			strings.Contains(strings.ToLower(k.File), q) {
			items = append(items, i18nItem{key: k, undefined: true})
		}
	}
	m.i18nItems = items
	if m.i18nCursor >= len(m.i18nItems) {
		m.i18nCursor = clampMin(0, len(m.i18nItems)-1)
	}
}

func (m *Model) rebuildCSSFiltered() {
	if m.cssResult == nil {
		m.cssItems = nil
		return
	}
	q := strings.ToLower(m.cssFilterQuery)
	var items []cssItem
	for _, c := range m.cssResult.Unused {
		if q == "" || strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.File), q) {
			items = append(items, cssItem{class: c, undefined: false})
		}
	}
	for _, c := range m.cssResult.Undefined {
		if q == "" || strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.File), q) {
			items = append(items, cssItem{class: c, undefined: true})
		}
	}
	m.cssItems = items
	if m.cssCursor >= len(m.cssItems) {
		m.cssCursor = clampMin(0, len(m.cssItems)-1)
	}
}
