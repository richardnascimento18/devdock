package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	gh "github.com/richardnascimento18/devdock/internal/github"
	"github.com/richardnascimento18/devdock/internal/pty"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
)

func TestPTYPostStepsAndFailure(t *testing.T) {
	root := t.TempDir()
	template := tmpl.Template{Name: "demo", PostSteps: []tmpl.TemplateStep{{Type: "builtin", Action: "touch", Path: "post"}}}
	p := newPTYScreen(100, 40, &template, root, tmpl.Vars{ProjectPath: root}, nil, root, gh.Repo{})
	for cmd := p.startNextStep(); cmd != nil; {
		msg := cmd().(ptyFlowMsg).msg
		p, cmd = p.Update(msg)
	}
	if !p.completed || p.exitErr != nil {
		t.Fatalf("post steps stalled: %+v", p)
	}
	if _, err := os.Stat(filepath.Join(root, "post")); err != nil {
		t.Fatal(err)
	}
	p = newPTYScreen(100, 40, &template, root, tmpl.Vars{ProjectPath: root}, []tmpl.TemplateStep{{Type: "builtin", Action: "touch", Path: "after"}}, root, gh.Repo{})
	next, cmd := p.Update(pty.ExitMsg{Err: errors.New("exit 7")})
	if !next.completed || next.exitErr == nil || cmd != nil {
		t.Fatal("failure advanced steps")
	}
}
func TestPTYControlBytesAndInputDoNotStartRead(t *testing.T) {
	p := newPTYScreen(100, 40, nil, t.TempDir(), tmpl.Vars{}, nil, "", gh.Repo{})
	p.ingestPTYData([]byte("hello\a\b\x00 world\n"))
	if len(p.vscreen.Lines()) == 0 {
		t.Fatal("output absent")
	}
	s, err := pty.NewSession([]string{"cat"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p.session = s
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd != nil {
		t.Fatal("input scheduled competing read")
	}
}
