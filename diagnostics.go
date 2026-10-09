package main

import (
	"github.com/richardnascimento18/devdock/internal/app"
	"github.com/richardnascimento18/devdock/internal/ui"
)

func diagnosticView(d app.Diagnostic) string {
	if d.Summary == "" {
		return ""
	}
	style := dimStyle
	switch d.Severity {
	case app.Error:
		style = errorStyle
	case app.Warning:
		style = warningStyle
	case app.Success:
		style = successStyle
	}
	return style.Render(ui.SafeBlock(d.Summary))
}
