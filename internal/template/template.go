package template

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/richardnascimento18/devdock/internal/core"
	"github.com/richardnascimento18/devdock/internal/fileutil"
)

type TemplateStep struct {
	Type   string `json:"type"`
	Run    string `json:"run"`
	Action string `json:"action"`
	Path   string `json:"path"`
	Shell  bool   `json:"shell,omitempty"`
	Output string `json:"output,omitempty"`
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
			{Type: "command", Run: ".venv/bin/pip freeze", Output: "requirements.txt"},
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
			return Clone(DefaultTemplates), writeErr
		}
		return Clone(DefaultTemplates), nil
	}
	if err != nil {
		return Clone(DefaultTemplates), fmt.Errorf("could not read templates.json: %w", err)
	}
	var tf TemplateFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return Clone(DefaultTemplates), fmt.Errorf(
			"templates.json contains invalid JSON:\n  %v\n\nfalling back to built-in templates", err,
		)
	}
	if errs := ValidateFile(tf); len(errs) > 0 {
		msg := "templates.json has errors:\n"
		for _, e := range errs {
			msg += "  - " + e + "\n"
		}
		return Clone(DefaultTemplates), fmt.Errorf("%s\nfalling back to built-in templates", msg)
	}
	return tf.Templates, nil
}

func writeDefaults(configDir string) error { return Save(configDir, DefaultTemplates) }

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
		for i, s := range append(append([]TemplateStep{}, t.Steps...), t.PostSteps...) {
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
				if !validRelative(s.Path, true) {
					errs = append(errs, fmt.Sprintf("template %q step %d: path %q must be relative and cannot traverse upward", t.Name, i, s.Path))
				}
			}
			if s.Type == "command" {
				if _, err := ParseCommand(s.Run); err != nil {
					errs = append(errs, fmt.Sprintf("template %q step %d: %v", t.Name, i, err))
				}
				if s.Output != "" && !validRelative(s.Output, true) {
					errs = append(errs, fmt.Sprintf("template %q: invalid output path", t.Name))
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

// validRelative requires a local path; destructive operations require a strict descendant.
func validRelative(rel string, strict bool) bool {
	for _, part := range strings.Split(filepath.FromSlash(rel), string(filepath.Separator)) {
		if part == ".." {
			return false
		}
	}
	return filepath.IsLocal(rel) && !strings.ContainsAny(rel, "\\\x00") && (!strict || filepath.Clean(rel) != ".")
}

func safeJoin(base, rel string) (string, error) {
	if !validRelative(rel, true) {
		return "", fmt.Errorf("path %q must be a strict descendant of the project directory", rel)
	}
	full := filepath.Join(base, rel)
	if err := core.ValidateDescendant(base, full); err != nil {
		return "", err
	}
	return full, nil
}

func ExecuteBuiltin(action, relPath, projectPath string) error {
	if _, err := safeJoin(projectPath, relPath); err != nil {
		return err
	}
	root, err := os.OpenRoot(projectPath)
	if err != nil {
		return err
	}
	defer root.Close()
	switch action {
	case "touch":
		if err := root.MkdirAll(filepath.Dir(relPath), 0o755); err != nil {
			return err
		}
		f, err := root.OpenFile(relPath, os.O_CREATE|os.O_RDONLY, 0o644)
		if err != nil {
			return err
		}
		return f.Close()
	case "mkdir":
		return root.MkdirAll(relPath, 0o755)
	case "rm":
		return root.RemoveAll(relPath)
	default:
		return fmt.Errorf("unknown builtin action: %s", action)
	}
}

// CommandOutput opens an explicitly declared output file within the project.
func CommandOutput(rel, projectPath string) (*os.File, error) {
	if _, err := safeJoin(projectPath, rel); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(projectPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.OpenFile(rel, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
}

func ExecuteSteps(steps []TemplateStep, workDir string, vars Vars, interactive bool) error {
	return ExecuteStepsContext(context.Background(), steps, workDir, vars, interactive)
}

func ExecuteStepsContext(ctx context.Context, steps []TemplateStep, workDir string, vars Vars, interactive bool) error {
	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch step.Type {
		case "builtin":
			if err := ExecuteBuiltin(step.Action, ExpandVars(step.Path, vars), vars.ProjectPath); err != nil {
				return fmt.Errorf("builtin %s: %w", step.Action, err)
			}
		case "command":
			args, err := CommandArgs(step, vars)
			if err != nil {
				return err
			}
			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Dir = workDir
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			var output *os.File
			if step.Output != "" {
				output, err = CommandOutput(ExpandVars(step.Output, vars), vars.ProjectPath)
				if err != nil {
					return err
				}
				cmd.Stdout = output
			}
			err = cmd.Run()
			if output != nil {
				err = errors.Join(err, output.Close())
			}
			if err != nil {
				return fmt.Errorf("command %q: %w", step.Run, err)
			}
		default:
			return fmt.Errorf("unknown step type %q", step.Type)
		}
	}
	return nil
}

func Run(t Template, domainPath, projectName string) (string, error) {
	if errs := ValidateFile(TemplateFile{Templates: []Template{t}}); len(errs) > 0 {
		return "", fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	vars := Vars{
		ProjectName: projectName,
		Domain:      filepath.Base(domainPath),
		Root:        filepath.Dir(domainPath),
	}
	projectPath, workDir, err := core.PrepareProject(vars.Root, vars.Domain, projectName, t.CreatesProjectFolder)
	if err != nil {
		return "", err
	}
	vars.ProjectPath = projectPath
	if err := ExecuteSteps(t.Steps, workDir, vars, t.Interactive); err != nil {
		return "", err
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
	return fileutil.WriteFileAtomic(filepath.Join(projectPath, ".devdock"), []byte(content), 0o644)
}

// Save validates the entire proposed collection before committing it.
func Save(configDir string, values []Template) error {
	if !filepath.IsAbs(configDir) {
		return fmt.Errorf("configuration directory must be absolute")
	}
	file := TemplateFile{Templates: values}
	if errs := ValidateFile(file); len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	data, err := json.MarshalIndent(file, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(Path(configDir), data, 0o644)
}

func Clone(src []Template) []Template {
	if src == nil {
		return nil
	}
	dst := append([]Template{}, src...)
	for i := range dst {
		dst[i].Steps = append([]TemplateStep(nil), src[i].Steps...)
		dst[i].PostSteps = append([]TemplateStep(nil), src[i].PostSteps...)
	}
	return dst
}
