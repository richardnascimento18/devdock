package main

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"
	uistate "github.com/richardnascimento18/devdock/internal/state"
	"os"
	"path/filepath"
	"strings"
)

func (m model) updateDeleteProject(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			typed := strings.TrimSpace(m.inputScr.input.Value())
			if typed != m.deleteTarget.Path {
				m.inputScr.err = "path does not match — try again or esc to cancel"
				m.inputScr.input.SetValue("")
				return m, nil
			}
			if err := core.ValidateDescendant(filepath.Join(m.deleteTarget.Root, m.deleteTarget.Domain), m.deleteTarget.Path); err != nil {
				m.inputScr.err = err.Error()
				return m, nil
			}
			return m.beginDelete(m.deleteTarget.Root, m.deleteTarget.Path)
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func (m model) updateDeleteDomain(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.pendingDomainName = ""
			m.pendingRoot = ""
			m.state = stateList
			return m, nil
		case "enter":
			if m.pendingDomainName == "" {
				typed := strings.TrimSpace(m.inputScr.input.Value())
				if typed == "" {
					m.inputScr.err = "name cannot be empty"
					return m, nil
				}
				roots := m.cfg.ActiveRoots()
				if !m.isAllMode() {
					roots = []string{m.activeRoot()}
				}
				foundRoot := ""
				for _, root := range roots {
					domains := m.workspaceDomains[root]
					for _, d := range domains {
						if d == typed {
							foundRoot = root
							break
						}
					}
					if foundRoot != "" {
						break
					}
				}
				if foundRoot == "" {
					m.inputScr.err = fmt.Sprintf("domain \"%s\" not found", typed)
					return m, nil
				}
				m.pendingDomainName = typed
				m.pendingRoot = foundRoot
				m.confirmDelDomain = newConfirmDeleteDomainScreen(filepath.Join(foundRoot, typed))
				return m, nil
			}
			typed := strings.TrimSpace(m.confirmDelDomain.input.Value())
			if typed != filepath.Join(m.pendingRoot, m.pendingDomainName) {
				m.confirmDelDomain.err = "name does not match — type exactly or esc to cancel"
				m.confirmDelDomain.input.SetValue("")
				return m, nil
			}
			return m.beginDelete(m.pendingRoot, filepath.Join(m.pendingRoot, m.pendingDomainName))
		}
	}
	if m.pendingDomainName == "" {
		var cmd tea.Cmd
		m.inputScr, cmd = m.inputScr.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.confirmDelDomain, cmd = m.confirmDelDomain.Update(msg)
	return m, cmd
}

func (m model) updateAddRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "enter":
			path := expandTilde(strings.TrimSpace(m.inputScr.input.Value()))
			if path == "" {
				m.inputScr.err = "path cannot be empty"
				return m, nil
			}
			path, err := filepath.Abs(path)
			if err != nil {
				m.inputScr.err = err.Error()
				return m, nil
			}
			info, err := os.Stat(path)
			if err != nil || !info.IsDir() {
				m.inputScr.err = "root must be an existing directory"
				return m, nil
			}
			proposed := m.cfg.Clone()
			proposed.AddRoot(path)
			committed, changed, err := config.Commit(m.cfg, proposed)
			if err != nil {
				m.inputScr.err = fmt.Sprintf("error saving config: %v", err)
				return m, nil
			}
			m.cfg = committed
			m.rootSel.SetRoots(m.cfg.ActiveRoots())
			if changed {
				m = m.rescan()
			}
			m.state = stateList
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.inputScr, cmd = m.inputScr.Update(msg)
	return m, cmd
}

func expandTilde(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func (m model) updateRemoveRoot(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.String() == "esc" {
			m.state = stateList
			return m, nil
		}
		if navPicker(&m.genericPicker, k.String()) {
			return m, nil
		}
		if k.String() == "enter" {
			if m.genericPicker.cursor < 0 || m.genericPicker.cursor >= len(m.cfg.ActiveRoots()) {
				m.state = stateList
				m.statusMsg = errorStyle.Render("no root selected")
				return m, nil
			}
			root := m.cfg.ActiveRoots()[m.genericPicker.cursor]
			proposed := m.cfg.Clone()
			proposed.RemoveRoot(root)
			proposedState := m.uiState.Clone()
			proposedState.RemovePath(root)
			if err := config.Save(proposed); err != nil {
				m.genericPicker.err = fmt.Sprintf("error saving config: %v", err)
				return m, nil
			}
			if err := uistate.Save(config.Dir(), proposedState); err != nil {
				// These are separate files: compensate before changing live state.
				rollback := config.Save(m.cfg)
				if rollback != nil {
					rollback = fmt.Errorf("configuration rollback failed: %w", rollback)
				}
				m.genericPicker.err = errors.Join(fmt.Errorf("root state cleanup failed: %w", err), rollback).Error()
				return m, nil
			}
			m.cfg = proposed
			m.uiState = proposedState
			m.collapsedNodes = proposedState.CollapsedNodes
			m.committedState = proposedState.Clone()
			m.persistenceErr = nil
			m.rootSel.SetRoots(m.cfg.ActiveRoots())
			m = m.rescan()
			m.state = stateList
			return m, nil
		}
	}
	return m, nil
}
