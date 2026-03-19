package core

// ProjectKind classifies how a directory was identified.
type ProjectKind string

const (
	KindProject  ProjectKind = "project"
	KindGroup    ProjectKind = "group"
	KindSubgroup ProjectKind = "subgroup"
)

// Project represents a local project on disk.
type Project struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Domain     string      `json:"domain"`
	Root       string      `json:"root"`
	GitHubRepo string      `json:"github_repo,omitempty"`
	Languages  []string    `json:"languages,omitempty"`
	Group      string      `json:"group,omitempty"`
	Subgroup   string      `json:"subgroup,omitempty"`
	Kind       ProjectKind `json:"kind,omitempty"`
}
