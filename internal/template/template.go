package template

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/fileutil"
)

type TemplateStep struct {
	Type   string `json:"type"`
	Run    string `json:"run"`
	Action string `json:"action"`
	Path   string `json:"path"`
	Shell  bool   `json:"shell,omitempty"`
}

type Template struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	Interactive          bool           `json:"interactive"`
	CreatesProjectFolder bool           `json:"creates_project_folder"`
	Layout               string         `json:"layout,omitempty"`
	Steps                []TemplateStep `json:"steps"`
	PostSteps            []TemplateStep `json:"post_steps,omitempty"`
}

type TemplateFile struct {
	Templates []Template `json:"templates"`
}

type Vars struct {
	ProjectName string
	ProjectPath string
	Domain      string
	Root        string
}

func Path(configDir string) string {
	return filepath.Join(configDir, "templates.json")
}

var DefaultTemplates = []Template{
	{
		Name:                 "Next.js",
		Description:          "Next.js app via create-next-app (interactive)",
		Interactive:          true,
		CreatesProjectFolder: true,
		Layout:               "walker",
		Steps:                []TemplateStep{{Type: "command", Run: "npx create-next-app@latest {{project_name}}"}},
		PostSteps:            []TemplateStep{{Type: "builtin", Action: "touch", Path: ".env.local"}},
	},
	{
		Name:                 "React + Vite",
		Description:          "React app scaffolded with Vite",
		Interactive:          true,
		CreatesProjectFolder: true,
		Layout:               "walker",
		Steps:                []TemplateStep{{Type: "command", Run: "npm create vite@latest {{project_name}} -- --template react-ts"}},
		PostSteps:            []TemplateStep{{Type: "builtin", Action: "touch", Path: ".env"}},
	},
	{
		Name:        "Go API",
		Description: "Go module with Gin HTTP framework",
		Layout:      "walker",
		Steps: []TemplateStep{
			{Type: "command", Run: "go mod init {{project_name}}"},
			{Type: "command", Run: "go get github.com/gin-gonic/gin"},
			{Type: "builtin", Action: "mkdir", Path: "cmd/server"},
			{Type: "builtin", Action: "touch", Path: "cmd/server/main.go"},
			{Type: "builtin", Action: "touch", Path: ".env"},
		},
	},
	{
		Name:        "Go CLI",
		Description: "Go module with Cobra CLI framework",
		Layout:      "walker",
		Steps: []TemplateStep{
			{Type: "command", Run: "go mod init {{project_name}}"},
			{Type: "command", Run: "go get github.com/spf13/cobra"},
			{Type: "builtin", Action: "mkdir", Path: "cmd"},
			{Type: "builtin", Action: "touch", Path: "cmd/root.go"},
			{Type: "builtin", Action: "touch", Path: "main.go"},
		},
	},
	{
		Name:        "Python FastAPI",
		Description: "Python project with FastAPI and uvicorn",
		Layout:      "walker",
		Steps: []TemplateStep{
			{Type: "command", Run: "python3 -m venv .venv"},
			{Type: "command", Run: ".venv/bin/pip install fastapi uvicorn python-dotenv"},
			{Type: "builtin", Action: "touch", Path: "main.py"},
			{Type: "builtin", Action: "touch", Path: ".env"},
			{Type: "builtin", Action: "touch", Path: "requirements.txt"},
			{Type: "command", Run: ".venv/bin/pip freeze > requirements.txt", Shell: true},
		},
	},
	{
		Name:        "Python Script",
		Description: "Minimal Python project with venv",
		Steps: []TemplateStep{
			{Type: "command", Run: "python3 -m venv .venv"},
			{Type: "builtin", Action: "touch", Path: "main.py"},
			{Type: "builtin", Action: "touch", Path: "requirements.txt"},
		},
	},
	{
		Name:                 "Rust",
		Description:          "Rust binary crate via cargo new",
		CreatesProjectFolder: true,
		Layout:               "walker",
		Steps:                []TemplateStep{{Type: "command", Run: "cargo new {{project_name}}"}},
	},
	{
		Name:                 "Rust Library",
		Description:          "Rust library crate via cargo new --lib",
		CreatesProjectFolder: true,
		Steps:                []TemplateStep{{Type: "command", Run: "cargo new --lib {{project_name}}"}},
	},
	{
		Name:        "Node.js",
		Description: "Bare Node.js project with npm init",
		Steps: []TemplateStep{
			{Type: "command", Run: "npm init -y"},
			{Type: "builtin", Action: "touch", Path: "index.js"},
			{Type: "builtin", Action: "touch", Path: ".env"},
		},
	},
	{
		Name:        "Static HTML",
		Description: "Plain HTML/CSS/JS website",
		Steps: []TemplateStep{
			{Type: "builtin", Action: "touch", Path: "index.html"},
			{Type: "builtin", Action: "touch", Path: "style.css"},
			{Type: "builtin", Action: "touch", Path: "script.js"},
			{Type: "builtin", Action: "mkdir", Path: "assets"},
		},
	},
}

func Load(configDir string) ([]Template, error) {
	data, err := os.ReadFile(Path(configDir))
	if os.IsNotExist(err) {
		if writeErr := writeDefaults(configDir); writeErr != nil {
			return DefaultTemplates, writeErr
		}
		return DefaultTemplates, nil
	}
	if err != nil {
		return DefaultTemplates, fmt.Errorf("could not read templates.json: %w", err)
	}
	var tf TemplateFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return DefaultTemplates, fmt.Errorf(
			"templates.json contains invalid JSON:\n  %v\n\nFalling back to built-in templates.", err,
		)
	}
	if errs := ValidateFile(tf); len(errs) > 0 {
		msg := "templates.json has errors:\n"
		for _, e := range errs {
			msg += "  - " + e + "\n"
		}
		return DefaultTemplates, fmt.Errorf("%s\nFalling back to built-in templates.", msg)
	}
	return tf.Templates, nil
}

func writeDefaults(configDir string) error {
	tf := TemplateFile{Templates: DefaultTemplates}
	data, err := json.MarshalIndent(tf, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(Path(configDir), data, 0o644)
}

func ValidateFile(tf TemplateFile) []string {
	var errs []string
	names := map[string]bool{}
	for _, t := range tf.Templates {
		if strings.TrimSpace(t.Name) == "" {
			errs = append(errs, "a template has an empty name")
			continue
		}
		if names[t.Name] {
			errs = append(errs, fmt.Sprintf("duplicate template name %q", t.Name))
		}
		names[t.Name] = true
		for i, s := range append(t.Steps, t.PostSteps...) {
			if s.Type != "command" && s.Type != "builtin" {
				errs = append(errs, fmt.Sprintf("template %q step %d: unknown type %q", t.Name, i, s.Type))
			}
			if s.Type == "builtin" {
				if s.Action != "touch" && s.Action != "mkdir" && s.Action != "rm" {
					errs = append(errs, fmt.Sprintf("template %q step %d: unknown builtin action %q", t.Name, i, s.Action))
				}
				if strings.TrimSpace(s.Path) == "" {
					errs = append(errs, fmt.Sprintf("template %q step %d: builtin step has empty path", t.Name, i))
				}
				if filepath.IsAbs(filepath.FromSlash(s.Path)) || strings.HasPrefix(s.Path, "..") {
					errs = append(errs, fmt.Sprintf("template %q step %d: path %q must be relative and cannot traverse upward", t.Name, i, s.Path))
				}
			}
			if s.Type == "command" && strings.TrimSpace(s.Run) == "" {
				errs = append(errs, fmt.Sprintf("template %q step %d: command step has empty run", t.Name, i))
			}
			if s.Type == "command" && s.Shell {
				errs = append(errs, fmt.Sprintf("template %q step %d: shell mode is not permitted in user-defined templates", t.Name, i))
			}
		}
	}
	return errs
}

func ExpandVars(s string, v Vars) string {
	s = strings.ReplaceAll(s, "{{project_name}}", v.ProjectName)
	s = strings.ReplaceAll(s, "{{project_path}}", v.ProjectPath)
	s = strings.ReplaceAll(s, "{{domain}}", v.Domain)
	s = strings.ReplaceAll(s, "{{root}}", v.Root)
	return s
}

// safeJoin joins base and rel and returns an error if the result escapes base.
func safeJoin(base, rel string) (string, error) {
	base = filepath.Clean(base)
	full := filepath.Clean(filepath.Join(base, filepath.FromSlash(rel)))
	if full != base && !strings.HasPrefix(full, base+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes the project directory", rel)
	}
	return full, nil
}

func ExecuteBuiltin(action, relPath, projectPath string) error {
	full, err := safeJoin(projectPath, relPath)
	if err != nil {
		return err
	}
	switch action {
	case "touch":
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(full, os.O_CREATE|os.O_RDONLY, 0o644)
		if err != nil {
			return err
		}
		return f.Close()
	case "mkdir":
		return os.MkdirAll(full, 0o755)
	case "rm":
		return os.RemoveAll(full)
	default:
		return fmt.Errorf("unknown builtin action: %s", action)
	}
}

func ExecuteSteps(steps []TemplateStep, workDir string, vars Vars, interactive bool) error {
	for _, step := range steps {
		switch step.Type {
		case "builtin":
			if err := ExecuteBuiltin(step.Action, ExpandVars(step.Path, vars), vars.ProjectPath); err != nil {
				return fmt.Errorf("builtin %s %q: %w", step.Action, step.Path, err)
			}
		case "command":
			cmdStr := ExpandVars(step.Run, vars)
			var cmd *exec.Cmd
			if step.Shell {
				cmd = exec.Command("sh", "-c", cmdStr)
			} else {
				args := strings.Fields(cmdStr)
				if len(args) == 0 {
					continue
				}
				cmd = exec.Command(args[0], args[1:]...)
			}
			cmd.Dir = workDir
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("command %q: %w", cmdStr, err)
			}
		}
	}
	return nil
}

func Run(t Template, domainPath, projectName string) (string, error) {
	vars := Vars{
		ProjectName: projectName,
		Domain:      filepath.Base(domainPath),
		Root:        filepath.Dir(domainPath),
	}
	var projectPath string
	if t.CreatesProjectFolder {
		vars.ProjectPath = filepath.Join(domainPath, projectName)
		projectPath = filepath.Join(domainPath, projectName)
		if err := ExecuteSteps(t.Steps, domainPath, vars, t.Interactive); err != nil {
			return "", err
		}
	} else {
		projectPath = filepath.Join(domainPath, projectName)
		if err := os.MkdirAll(projectPath, 0o755); err != nil {
			return "", fmt.Errorf("could not create project directory: %w", err)
		}
		vars.ProjectPath = projectPath
		if err := ExecuteSteps(t.Steps, projectPath, vars, t.Interactive); err != nil {
			return "", err
		}
	}
	if len(t.PostSteps) > 0 {
		if err := ExecuteSteps(t.PostSteps, projectPath, vars, false); err != nil {
			return "", fmt.Errorf("post-step failed: %w", err)
		}
	}
	return projectPath, nil
}

func WriteDevDockMarkerFile(projectPath string) error {
	content := "type = \"project\"\n"
	return os.WriteFile(filepath.Join(projectPath, ".devdock"), []byte(content), 0o644)
}
