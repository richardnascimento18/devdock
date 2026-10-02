package core

// ProjectKind describes a local discovery boundary.
type ProjectKind string

const (
	KindProject ProjectKind = "project"
	KindGroup   ProjectKind = "group"
)

// Project separates the display name from the directory and workspace location.
type Project struct {
	Location
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	GitHubRepo string      `json:"github_repo,omitempty"`
	Languages  []string    `json:"languages,omitempty"`
	Kind       ProjectKind `json:"kind,omitempty"`
}

type ProjectKey string

func (p Project) Key() ProjectKey { return ProjectKey(p.Path) }
