package main

import (
	"os"
	"path/filepath"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/charmbracelet/bubbles/list"
)

func (m model) buildListItems(projects []core.Project, verified bool) []list.Item {
	showRoot := m.isAllMode() && len(m.cfg.ActiveRoots()) > 1

	if !m.treeMode {
		items := make([]list.Item, len(projects))
		for i, p := range projects {
			it := item{project: p, showRoot: showRoot, verified: verified, indent: 0}
			items[i] = flatItem{item: it}
		}
		items = append(items, m.buildEmptyGroupFlatItems(showRoot)...)
		return items
	}

	type subgroupState struct {
		name     string
		projects []core.Project
	}
	type groupState struct {
		name       string
		domain     string
		root       string
		totalCount int
		subgroups  map[string]*subgroupState
		subOrder   []string
		projects   []core.Project
	}
	type domainState struct {
		root       string
		name       string
		groups     map[string]*groupState
		groupOrder []string
		ungrouped  []core.Project
	}

	domains := make(map[string]*domainState)
	var domainOrder []string

	for _, p := range projects {
		dk := p.Root + "::" + p.Domain
		if _, ok := domains[dk]; !ok {
			domains[dk] = &domainState{
				root:   p.Root,
				name:   p.Domain,
				groups: make(map[string]*groupState),
			}
			domainOrder = append(domainOrder, dk)
		}
		d := domains[dk]

		if p.Group == "" {
			d.ungrouped = append(d.ungrouped, p)
			continue
		}

		gk := p.Group
		if _, ok := d.groups[gk]; !ok {
			d.groups[gk] = &groupState{
				name:      p.Group,
				domain:    p.Domain,
				root:      p.Root,
				subgroups: make(map[string]*subgroupState),
			}
			d.groupOrder = append(d.groupOrder, gk)
		}
		g := d.groups[gk]
		g.totalCount++

		if p.Subgroup == "" {
			g.projects = append(g.projects, p)
			continue
		}
		sk := p.Subgroup
		if _, ok := g.subgroups[sk]; !ok {
			g.subgroups[sk] = &subgroupState{name: p.Subgroup}
			g.subOrder = append(g.subOrder, sk)
		}
		g.subgroups[sk].projects = append(g.subgroups[sk].projects, p)
	}

	// Register empty groups — directories with .ddgroup but no projects.
	for _, root := range m.cfg.ActiveRoots() {
		if !m.isAllMode() && root != m.activeRoot() {
			continue
		}
		domainEntries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, de := range domainEntries {
			if !de.IsDir() {
				continue
			}
			domain := de.Name()
			domainPath := filepath.Join(root, domain)
			emptyGroups := core.ScanGroupsInDomain(domainPath, classifyFn)
			for _, gi := range emptyGroups {
				dk := root + "::" + domain
				if _, ok := domains[dk]; !ok {
					domains[dk] = &domainState{
						root:   root,
						name:   domain,
						groups: make(map[string]*groupState),
					}
					domainOrder = append(domainOrder, dk)
				}
				d := domains[dk]
				if _, ok := d.groups[gi.Name]; !ok {
					d.groups[gi.Name] = &groupState{
						name:      gi.Name,
						domain:    domain,
						root:      root,
						subgroups: make(map[string]*subgroupState),
					}
					d.groupOrder = append(d.groupOrder, gi.Name)
				}
			}
		}
	}

	var items []list.Item

	for _, dk := range domainOrder {
		d := domains[dk]

		for _, p := range d.ungrouped {
			items = append(items, item{project: p, showRoot: showRoot, verified: verified, indent: 0})
		}

		for _, gk := range d.groupOrder {
			g := d.groups[gk]
			gKey := groupKey(g.root, g.domain, g.name)
			collapsed := m.collapsedGroups[gKey]

			shownCount := len(g.projects)
			if !collapsed {
				for _, sk := range g.subOrder {
					sg := g.subgroups[sk]
					sgKey := subgroupKey(g.root, g.domain, g.name, sg.name)
					if !m.collapsedSubgroups[sgKey] {
						shownCount += len(sg.projects)
					}
				}
			}

			isEmpty := g.totalCount == 0
			gi := groupItem{
				groupKey:   gKey,
				name:       g.name,
				domain:     g.domain,
				root:       g.root,
				collapsed:  collapsed,
				totalCount: g.totalCount,
				shownCount: shownCount,
				empty:      isEmpty,
			}
			items = append(items, gi)

			if collapsed || isEmpty {
				continue
			}

			for _, p := range g.projects {
				items = append(items, item{project: p, showRoot: showRoot, verified: verified, indent: 1})
			}

			for _, sk := range g.subOrder {
				sg := g.subgroups[sk]
				sgKey := subgroupKey(g.root, g.domain, g.name, sg.name)
				sgCollapsed := m.collapsedSubgroups[sgKey]

				sgi := subgroupItem{
					subgroupKey: sgKey,
					name:        sg.name,
					parentGroup: g.name,
					domain:      g.domain,
					root:        g.root,
					collapsed:   sgCollapsed,
					totalCount:  len(sg.projects),
					shownCount:  len(sg.projects),
				}
				if sgCollapsed {
					sgi.shownCount = 0
				}
				items = append(items, sgi)

				if !sgCollapsed {
					for _, p := range sg.projects {
						items = append(items, item{project: p, showRoot: showRoot, verified: verified, indent: 2})
					}
				}
			}
		}
	}

	return items
}

func (m model) buildEmptyGroupFlatItems(showRoot bool) []list.Item {
	var items []list.Item
	for _, root := range m.cfg.ActiveRoots() {
		if !m.isAllMode() && root != m.activeRoot() {
			continue
		}
		domainEntries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, de := range domainEntries {
			if !de.IsDir() {
				continue
			}
			domain := de.Name()
			domainPath := filepath.Join(root, domain)
			groups := core.ScanGroupsInDomain(domainPath, classifyFn)
			for _, gi := range groups {
				groupPath := filepath.Join(domainPath, gi.Name)
				if !groupHasProjects(groupPath) {
					items = append(items, emptyGroupFlatItem{
						name:     gi.Name,
						domain:   domain,
						root:     root,
						showRoot: showRoot,
					})
				}
			}
		}
	}
	return items
}

func groupHasProjects(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			if classifyFn(filepath.Join(path, e.Name()), 0) == core.KindProject {
				return true
			}
		}
	}
	return false
}

