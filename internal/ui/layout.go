package ui

// Minimum useful pane widths include their title and content, not borders.
const (
	MinWidth     = 24
	MinHeight    = 8
	WorkspaceMin = 24
	ProjectsMin  = 40
	InspectorMin = 30
	PaneGap      = 2
	ChromeHeight = 6
)

type Mode uint8

const (
	Tiny Mode = iota
	Narrow
	Medium
	Wide
)

type Layout struct {
	Mode                           Mode
	Width, Height, BodyHeight      int
	Workspace, Projects, Inspector int
}

func Measure(width, height int) Layout {
	l := Layout{Width: max(width, 0), Height: max(height, 0)}
	if width < MinWidth || height < MinHeight {
		return l
	}
	l.BodyHeight = max(height-ChromeHeight, 1)
	switch {
	case width >= WorkspaceMin+ProjectsMin+InspectorMin+2*PaneGap:
		l.Mode = Wide
		l.Workspace = min(30, max(WorkspaceMin, width/5))
		l.Inspector = min(44, max(InspectorMin, width/3))
		l.Inspector = min(l.Inspector, width-l.Workspace-ProjectsMin-2*PaneGap)
		l.Projects = width - l.Workspace - l.Inspector - 2*PaneGap
	case width >= WorkspaceMin+ProjectsMin+PaneGap:
		l.Mode = Medium
		l.Workspace = min(30, max(WorkspaceMin, width/4))
		l.Projects = width - l.Workspace - PaneGap
	default:
		l.Mode = Narrow
		l.Workspace, l.Projects, l.Inspector = width, width, width
	}
	return l
}

type Pane uint8

const (
	Workspace Pane = iota
	Projects
	Inspector
)

func (p Pane) String() string {
	switch p {
	case Workspace:
		return "Workspace"
	case Inspector:
		return "Inspector"
	default:
		return "Projects"
	}
}

func (p Pane) Cycle(delta int, inspector bool) Pane {
	count := 2
	if inspector {
		count = 3
	}
	return Pane((int(p) + delta + count) % count)
}
