package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/config"
	"github.com/richardnascimento18/devdock/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) startCreateGroup() (tea.Model, tea.Cmd) {
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		m.statusMsg = errorStyle.Render("no roots configured")
		return m, nil
	}
	// Determine which root/domain to use from context
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openDomainPickerForGroup()
	}
	opts := make([]string, len(roots))
	for i, r := range roots {
		opts[i] = config.RootName(r)
	}
	m.genericPicker = newGenericPicker("Create Group — select root:", opts, "↑/↓  •  enter  •  esc")
	m.state = statePickRoot
	// reuse statePickRoot but we need to differentiate; use a sentinel
	m.pendingProjectName = "__creategroup__"
	return m, nil
}

func (m model) openDomainPickerForGroup() (tea.Model, tea.Cmd) {
	domains, _ := core.ScanDomainsInRoot(m.pendingRoot)
	if len(domains) == 0 {
		m.statusMsg = errorStyle.Render("no domains in this root — create a domain first (N)")
		return m, nil
	}
	opts := make([]string, len(domains))
	copy(opts, domains)
	m.genericPicker = newGenericPicker("Create Group — select domain:", opts, "↑/↓  •  enter  •  esc")
	m.state = stateCreateGroup
	return m, nil
}

func (m model) updateCreateGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.pendingDomain = ""
			m.state = stateList
			return m, nil
		}

		// Phase 1: picking domain (genericPicker is active)
		if m.pendingDomain == "" {
			switch k.String() {
			case "up", "k":
				if m.genericPicker.cursor > 0 {
					m.genericPicker.cursor--
				}
				return m, nil
			case "down", "j":
				if m.genericPicker.cursor < len(m.genericPicker.options)-1 {
					m.genericPicker.cursor++
				}
				return m, nil
			case "enter":
				m.pendingDomain = m.genericPicker.options[m.genericPicker.cursor]
				m.inputScr = newInputScreen(
					fmt.Sprintf("New Group in \"%s/%s\" — enter name:", config.RootName(m.pendingRoot), m.pendingDomain),
					"group-name", "enter confirm  •  esc cancel",
				)
				return m, nil
			}
			return m, nil
		}

		// Phase 2: entering group name (inputScr is active)
		if k.String() == "enter" {
			name := strings.TrimSpace(m.inputScr.input.Value())
			if name == "" {
				m.inputScr.err = "name cannot be empty"
				return m, nil
			}
			if !isSafePathName(name) {
				m.inputScr.err = "name contains invalid characters (/ and \\ are not allowed)"
				return m, nil
			}
			groupPath := filepath.Join(m.pendingRoot, m.pendingDomain, name)
			if err := os.MkdirAll(groupPath, 0o755); err != nil {
				m.inputScr.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			if err := os.WriteFile(filepath.Join(groupPath, ".ddgroup"), []byte{}, 0o644); err != nil {
				m.inputScr.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			m.pendingDomain = ""
			m = m.rescan()
			m.state = stateList
			m.statusMsg = successStyle.Render(fmt.Sprintf("✓  group \"%s\" created", name))
			return m, nil
		}
	}

	// Phase 2: delegate to inputScr
	if m.pendingDomain != "" {
		var cmd tea.Cmd
		m.inputScr, cmd = m.inputScr.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) startDeleteGroup() (tea.Model, tea.Cmd) {
	roots := m.cfg.ActiveRoots()
	if len(roots) == 0 {
		m.statusMsg = errorStyle.Render("no roots configured")
		return m, nil
	}
	if len(roots) == 1 {
		m.pendingRoot = roots[0]
		return m.openGroupPickerForDelete()
	}
	opts := make([]string, len(roots))
	for i, r := range roots {
		opts[i] = config.RootName(r)
	}
	m.genericPicker = newGenericPicker("Delete Group — select root:", opts, "↑/↓  •  enter  •  esc")
	m.pendingProjectName = "__deletegroup__"
	m.state = statePickRoot
	return m, nil
}

func (m model) openGroupPickerForDelete() (tea.Model, tea.Cmd) {
	domains, _ := core.ScanDomainsInRoot(m.pendingRoot)
	var groups []string
	for _, d := range domains {
		domainPath := filepath.Join(m.pendingRoot, d)
		gs := core.ScanGroupsInDomain(domainPath, classifyFn)
		for _, g := range gs {
			groups = append(groups, d+"/"+g.Name)
		}
	}
	if len(groups) == 0 {
		m.statusMsg = errorStyle.Render("no groups found in this root")
		m.state = stateList
		return m, nil
	}
	m.genericPicker = newGenericPicker("Delete Group — select group:", groups, "↑/↓  •  enter  •  esc cancel")
	m.state = stateDeleteGroup
	return m, nil
}

func (m model) updateDeleteGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			m.state = stateList
			return m, nil
		case "up", "k":
			if m.genericPicker.cursor > 0 {
				m.genericPicker.cursor--
			}
			return m, nil
		case "down", "j":
			if m.genericPicker.cursor < len(m.genericPicker.options)-1 {
				m.genericPicker.cursor++
			}
			return m, nil
		case "enter":
			chosen := m.genericPicker.options[m.genericPicker.cursor]
			parts := strings.SplitN(chosen, "/", 2)
			if len(parts) != 2 {
				return m, nil
			}
			domain, groupName := parts[0], parts[1]
			groupPath := filepath.Join(m.pendingRoot, domain, groupName)
			if !core.IsDescendant(m.pendingRoot, groupPath) {
				m.genericPicker.err = "refusing to delete: path is outside configured root"
				return m, nil
			}
			if err := os.RemoveAll(groupPath); err != nil {
				m.genericPicker.err = fmt.Sprintf("error: %v", err)
				return m, nil
			}
			m = m.rescan()
			m.state = stateList
			m.statusMsg = successStyle.Render(fmt.Sprintf("✓  group \"%s\" deleted", groupName))
			return m, nil
		}
	}
	return m, nil
}
