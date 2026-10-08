package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/richardnascimento18/devdock/internal/app"
)

func TestDiagnosticsStayPlainAndRenderAtBoundary(t *testing.T) {
	m := fixtureModel(t, t.TempDir())
	m.diagnostic = app.Diagnostic{Severity: app.Error, Summary: "failed\x1b]52;c;payload\a", Operation: "save"}
	if strings.Contains(m.diagnostic.Summary, "\x1b[31") {
		t.Fatal("styled state")
	}
	assertSafeDisplay(t, m.View())
	view := diagnosticView(m.diagnostic)
	if !strings.Contains(ansi.Strip(view), "failed␛]52") || m.diagnostic.Operation != "save" {
		t.Fatal("lost error details")
	}
}
