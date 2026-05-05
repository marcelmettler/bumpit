package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/maece/bumpit/internal/pkg"
)

func newDetailModel() *Model {
	m := New(".")
	m.state = stateDetail
	m.width = 120
	m.height = 40
	m.packages = []*pkg.PackageUpdate{
		{Name: "react", Current: "18.2.0", Latest: "19.0.0", Kind: pkg.KindMajor,
			Changelog: "## v19\n\n- something\n", ChangelogFetched: true,
			IsEligible: true, DirName: "root"},
	}
	m.filtered = m.packages
	m.detailPackage = m.packages[0]
	m.detailScroll = 0
	return m
}

func key(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "j":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	case "k":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestDetailEsc(t *testing.T) {
	m := newDetailModel()
	newM, _ := m.Update(key("esc"))
	result := newM.(*Model)
	if result.state != stateList {
		t.Errorf("after esc: got state %d, want stateList (%d)", result.state, stateList)
	}
	if result.detailPackage != nil {
		t.Error("after esc: detailPackage should be nil")
	}
}

func TestDetailScrollDown(t *testing.T) {
	m := newDetailModel()
	newM, _ := m.Update(key("j"))
	result := newM.(*Model)
	if result.detailScroll != 1 {
		t.Errorf("after j: got detailScroll %d, want 1", result.detailScroll)
	}
}

func TestDetailScrollUp(t *testing.T) {
	m := newDetailModel()
	m.detailScroll = 3
	newM, _ := m.Update(key("k"))
	result := newM.(*Model)
	if result.detailScroll != 2 {
		t.Errorf("after k: got detailScroll %d, want 2", result.detailScroll)
	}
}

func TestDetailSpace(t *testing.T) {
	m := newDetailModel()
	m.detailPackage.Selected = false
	newM, _ := m.Update(key(" "))
	result := newM.(*Model)
	if !result.detailPackage.Selected {
		t.Error("after space: package should be selected")
	}
}
