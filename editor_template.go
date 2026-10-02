package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tmpl "github.com/richardnascimento18/devdock/internal/template"
	"path/filepath"
	"strings"
)

type templateEditorLayer int

const (
	telStepList   templateEditorLayer = iota
	telStepEdit                       // editing a single step
	telMetaEdit                       // editing name/description/flags
	telConfirmDel                     // confirm delete step
)

type stepDraft struct {
	stepType string // "command" or "builtin"
	run      string
	action   string
	path     string
	output   string
	shell    bool
	isPost   bool // belongs to post_steps
}

type templateEditor struct {
	originalName string
	isNew        bool
	layout       string

	nameInput            textinput.Model
	descInput            textinput.Model
	interactive          bool
	createsProjectFolder bool

	steps  []stepDraft
	cursor int
	layer  templateEditorLayer

	// step edit fields
	editIdx        int
	editTypeCursor int // 0=command, 1=builtin
	editRunInput   textinput.Model
	editActCursor  int // 0=touch,1=mkdir,2=rm
	editPathInput  textinput.Model
	editIsPost     bool
	editFocus      int // which field is focused

	// meta edit
	metaFocus int // 0=name, 1=desc, 2=interactive, 3=createsFolder

	// typing=true means keystrokes go to the focused text input;
	// j/k/navigation only work when typing=false.
	typing bool

	statusMsg string
	result    editorResult
}

var builtinActions = []string{"touch", "mkdir", "rm"}

func newTemplateEditor(t tmpl.Template, isNew bool) templateEditor {
	nameInp := newSmallInput(t.Name, "template name", 40)
	descInp := newSmallInput(t.Description, "short description", 60)

	var steps []stepDraft
	for _, s := range t.Steps {
		steps = append(steps, stepDraftFromTemplate(s, false))
	}
	for _, s := range t.PostSteps {
		steps = append(steps, stepDraftFromTemplate(s, true))
	}

	return templateEditor{
		originalName:         t.Name,
		layout:               t.Layout,
		isNew:                isNew,
		nameInput:            nameInp,
		descInput:            descInp,
		interactive:          t.Interactive,
		createsProjectFolder: t.CreatesProjectFolder,
		steps:                steps,
	}
}

func stepDraftFromTemplate(s tmpl.TemplateStep, isPost bool) stepDraft {
	return stepDraft{
		stepType: s.Type,
		run:      s.Run,
		action:   s.Action,
		path:     s.Path,
		shell:    s.Shell,
		output:   s.Output,
		isPost:   isPost,
	}
}

func (te *templateEditor) toTemplate() tmpl.Template {
	t := tmpl.Template{
		Name:                 strings.TrimSpace(te.nameInput.Value()),
		Layout:               te.layout,
		Description:          strings.TrimSpace(te.descInput.Value()),
		Interactive:          te.interactive,
		CreatesProjectFolder: te.createsProjectFolder,
	}
	for _, sd := range te.steps {
		step := tmpl.TemplateStep{
			Type:   sd.stepType,
			Run:    sd.run,
			Action: sd.action,
			Path:   sd.path,
			Shell:  sd.shell,
			Output: sd.output,
		}
		if sd.isPost {
			t.PostSteps = append(t.PostSteps, step)
		} else {
			t.Steps = append(t.Steps, step)
		}
	}
	return t
}

func (te templateEditor) Update(msg tea.Msg) (templateEditor, tea.Cmd) {
	switch te.layer {
	case telStepEdit:
		return te.updateStepEdit(msg)
	case telMetaEdit:
		return te.updateMetaEdit(msg)
	case telConfirmDel:
		return te.updateConfirmDel(msg)
	default:
		return te.updateStepList(msg)
	}
}

func (te templateEditor) updateStepList(msg tea.Msg) (templateEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return te, nil
	}
	listLen := len(te.steps) + 2 // +2: "add step", "add post-step"
	switch k.String() {
	case "ctrl+c", "esc":
		te.result = editorResultCancel
	case "ctrl+s":
		te.result = editorResultSave
	case "j", "down":
		if te.cursor < listLen-1 {
			te.cursor++
		}
	case "k", "up":
		if te.cursor > 0 {
			te.cursor--
		}
	case "m":
		// edit meta — start in navigation mode, no input focused
		te.metaFocus = 0
		te.typing = false
		te.nameInput.Blur()
		te.descInput.Blur()
		te.layer = telMetaEdit
	case "enter":
		te.statusMsg = ""
		if te.cursor < len(te.steps) {
			te.editIdx = te.cursor
			te.openStepEdit(te.steps[te.cursor])
		} else if te.cursor == len(te.steps) {
			// add normal step
			te.steps = append(te.steps, stepDraft{stepType: "command"})
			te.editIdx = len(te.steps) - 1
			te.openStepEdit(te.steps[te.editIdx])
		} else {
			// add post step
			te.steps = append(te.steps, stepDraft{stepType: "command", isPost: true})
			te.editIdx = len(te.steps) - 1
			te.openStepEdit(te.steps[te.editIdx])
		}
	case "d", "x":
		if te.cursor < len(te.steps) {
			te.layer = telConfirmDel
		}
	}
	return te, nil
}

func (te *templateEditor) openStepEdit(sd stepDraft) {
	te.editTypeCursor = 0
	if sd.stepType == "builtin" {
		te.editTypeCursor = 1
	}
	te.editRunInput = newSmallInput(sd.run, "command string e.g. go mod init {{project_name}}", 60)
	te.editActCursor = 0
	for i, a := range builtinActions {
		if a == sd.action {
			te.editActCursor = i
			break
		}
	}
	te.editPathInput = newSmallInput(sd.path, "relative path e.g. cmd/main.go", 50)
	te.editIsPost = sd.isPost
	te.editFocus = 0
	te.typing = false // always start in navigation mode
	te.layer = telStepEdit
}

func (te templateEditor) updateStepEdit(msg tea.Msg) (templateEditor, tea.Cmd) {
	// isTextFocus returns true when the cursor is on a field that has a text
	// input — those are the only fields that benefit from typing mode.
	isTextFocus := func() bool {
		return (te.editTypeCursor == 0 && te.editFocus == 1) ||
			(te.editTypeCursor == 1 && te.editFocus == 2)
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if te.typing {
			return te.forwardStepInput(msg)
		}
		return te, nil
	}

	// --- Typing mode: all keys go straight to the focused input. ---
	if te.typing {
		switch k.String() {
		case "esc":
			te.typing = false
			te.editRunInput.Blur()
			te.editPathInput.Blur()
			return te, nil
		case "ctrl+s":
			te.result = editorResultSave
			return te, nil
		case "ctrl+c":
			te.result = editorResultCancel
			return te, nil
		}
		return te.forwardStepInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "ctrl+c":
		te.result = editorResultCancel
		return te, nil
	case "esc":
		// discard this edit, revert steps if it was a new append
		if te.editIdx == len(te.steps)-1 && te.steps[te.editIdx].run == "" && te.steps[te.editIdx].path == "" {
			te.steps = te.steps[:te.editIdx]
		}
		te.layer = telStepList
		return te, nil
	case "ctrl+s":
		te.result = editorResultSave
		return te, nil
	case "i":
		if isTextFocus() {
			te.typing = true
			te.syncStepEditFocus()
		}
		return te, nil
	case "tab", "down", "j":
		te.editFocus = (te.editFocus + 1) % te.editFieldCount()
		te.syncStepEditFocus()
		return te, nil
	case "shift+tab", "up", "k":
		te.editFocus = (te.editFocus - 1 + te.editFieldCount()) % te.editFieldCount()
		te.syncStepEditFocus()
		return te, nil
	case "left", "h":
		if te.editFocus == 0 {
			te.editTypeCursor = (te.editTypeCursor + 1) % 2
			te.syncStepEditFocus()
			return te, nil
		}
		if te.editFocus == 1 && te.editTypeCursor == 1 {
			te.editActCursor = (te.editActCursor + len(builtinActions) - 1) % len(builtinActions)
			return te, nil
		}
	case "right", "l":
		if te.editFocus == 0 {
			te.editTypeCursor = (te.editTypeCursor + 1) % 2
			te.syncStepEditFocus()
			return te, nil
		}
		if te.editFocus == 1 && te.editTypeCursor == 1 {
			te.editActCursor = (te.editActCursor + 1) % len(builtinActions)
			return te, nil
		}
	case "enter", " ":
		lastField := te.editFieldCount() - 1
		if te.editFocus == lastField {
			te.editIsPost = !te.editIsPost
			return te, nil
		}
		return te.commitStepEdit()
	}
	return te, nil
}

// forwardStepInput routes a message to whichever text input is active.
func (te templateEditor) forwardStepInput(msg tea.Msg) (templateEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch te.editTypeCursor {
	case 0:
		if te.editFocus == 1 {
			te.editRunInput, cmd = te.editRunInput.Update(msg)
		}
	case 1:
		if te.editFocus == 2 {
			te.editPathInput, cmd = te.editPathInput.Update(msg)
		}
	}
	return te, cmd
}

func (te templateEditor) editFieldCount() int {
	if te.editTypeCursor == 0 {
		return 3 // type, run, isPost
	}
	return 4 // type, action, path, isPost
}

func (te *templateEditor) syncStepEditFocus() {
	te.editRunInput.Blur()
	te.editPathInput.Blur()
	if !te.typing {
		return
	}
	if te.editTypeCursor == 0 && te.editFocus == 1 {
		te.editRunInput.Focus()
	}
	if te.editTypeCursor == 1 && te.editFocus == 2 {
		te.editPathInput.Focus()
	}
}

func (te templateEditor) commitStepEdit() (templateEditor, tea.Cmd) {
	sd := &te.steps[te.editIdx]
	if te.editTypeCursor == 0 {
		run := strings.TrimSpace(te.editRunInput.Value())
		if run == "" {
			te.statusMsg = errorStyle.Render("✗  command cannot be empty")
			return te, nil
		}
		sd.stepType = "command"
		sd.run = run
		sd.shell = false
		sd.action = ""
		sd.path = ""
	} else {
		path := strings.TrimSpace(te.editPathInput.Value())
		if path == "" {
			te.statusMsg = errorStyle.Render("✗  path cannot be empty")
			return te, nil
		}
		if filepath.IsAbs(filepath.FromSlash(path)) || strings.HasPrefix(path, "..") {
			te.statusMsg = errorStyle.Render("✗  path must be relative and cannot traverse upward")
			return te, nil
		}
		sd.stepType = "builtin"
		sd.action = builtinActions[te.editActCursor]
		sd.path = path
		sd.run = ""
	}
	sd.isPost = te.editIsPost
	te.statusMsg = ""
	te.layer = telStepList
	return te, nil
}

func (te templateEditor) updateMetaEdit(msg tea.Msg) (templateEditor, tea.Cmd) {
	isTextFocus := te.metaFocus == 0 || te.metaFocus == 1

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if te.typing && isTextFocus {
			return te.forwardMetaInput(msg)
		}
		return te, nil
	}

	// --- Typing mode. ---
	if te.typing {
		switch k.String() {
		case "esc":
			te.typing = false
			te.nameInput.Blur()
			te.descInput.Blur()
			return te, nil
		case "ctrl+s":
			te.result = editorResultSave
			return te, nil
		case "ctrl+c":
			te.result = editorResultCancel
			return te, nil
		}
		return te.forwardMetaInput(msg)
	}

	// --- Navigation mode. ---
	switch k.String() {
	case "ctrl+c":
		te.result = editorResultCancel
		return te, nil
	case "esc":
		te.layer = telStepList
		return te, nil
	case "ctrl+s":
		te.result = editorResultSave
		return te, nil
	case "i":
		if isTextFocus {
			te.typing = true
			te.syncMetaFocus()
		}
		return te, nil
	case "tab", "j", "down":
		te.metaFocus = (te.metaFocus + 1) % 4
		te.syncMetaFocus()
		return te, nil
	case "shift+tab", "k", "up":
		te.metaFocus = (te.metaFocus - 1 + 4) % 4
		te.syncMetaFocus()
		return te, nil
	case "enter", " ":
		switch te.metaFocus {
		case 2:
			te.interactive = !te.interactive
		case 3:
			te.createsProjectFolder = !te.createsProjectFolder
		}
		return te, nil
	}
	return te, nil
}

// forwardMetaInput routes a message to whichever meta text input is active.
func (te templateEditor) forwardMetaInput(msg tea.Msg) (templateEditor, tea.Cmd) {
	var cmd tea.Cmd
	switch te.metaFocus {
	case 0:
		te.nameInput, cmd = te.nameInput.Update(msg)
	case 1:
		te.descInput, cmd = te.descInput.Update(msg)
	}
	return te, cmd
}

func (te *templateEditor) syncMetaFocus() {
	te.nameInput.Blur()
	te.descInput.Blur()
	if !te.typing {
		return
	}
	switch te.metaFocus {
	case 0:
		te.nameInput.Focus()
	case 1:
		te.descInput.Focus()
	}
}

func (te templateEditor) updateConfirmDel(msg tea.Msg) (templateEditor, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return te, nil
	}
	switch k.String() {
	case "y", "enter":
		te.steps = append(te.steps[:te.cursor], te.steps[te.cursor+1:]...)
		if te.cursor >= len(te.steps) {
			te.cursor = len(te.steps) - 1
		}
		if te.cursor < 0 {
			te.cursor = 0
		}
		te.layer = telStepList
	case "n", "esc":
		te.layer = telStepList
	}
	return te, nil
}

// ---------------------------------------------------------------------------
// templateEditor.View
// ---------------------------------------------------------------------------

func (te templateEditor) View(w, h int) string {
	switch te.layer {
	case telStepEdit:
		return te.viewStepEdit(w, h)
	case telMetaEdit:
		return te.viewMetaEdit(w, h)
	case telConfirmDel:
		return te.viewConfirmDel(w, h)
	default:
		return te.viewStepList(w, h)
	}
}

func (te templateEditor) viewStepList(w, h int) string {
	title := "Edit Template"
	if te.isNew {
		title = "New Template"
	}

	var inner strings.Builder
	name := strings.TrimSpace(te.nameInput.Value())
	if name == "" {
		name = "(unnamed)"
	}
	inner.WriteString(promptStyle.Render("Template: ") +
		lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(name) +
		dimStyle.Render("  (m to edit meta)") + "\n\n")

	inner.WriteString(promptStyle.Render("Steps:") + "\n")
	for i, sd := range te.steps {
		label := stepLabel(sd)
		postTag := ""
		if sd.isPost {
			postTag = dimStyle.Render(" [post]")
		}
		if i == te.cursor {
			inner.WriteString(activeStyle.Render("▶ "+label) + postTag + "\n")
		} else {
			inner.WriteString("  " + lipgloss.NewStyle().Foreground(theme.Secondary).Render(label) + postTag + "\n")
		}
	}

	addIdx := len(te.steps)
	addPostIdx := len(te.steps) + 1
	addLabel := lipgloss.NewStyle().Foreground(theme.Info).Bold(true).Render("✦  add step")
	addPostLabel := lipgloss.NewStyle().Foreground(theme.Info).Bold(true).Render("✦  add post-step")

	if te.cursor == addIdx {
		inner.WriteString(activeStyle.Render("▶ ") + addLabel + "\n")
	} else {
		inner.WriteString("  " + addLabel + "\n")
	}
	if te.cursor == addPostIdx {
		inner.WriteString(activeStyle.Render("▶ ") + addPostLabel + "\n")
	} else {
		inner.WriteString("  " + addPostLabel + "\n")
	}

	if te.statusMsg != "" {
		inner.WriteString("\n" + te.statusMsg)
	}
	inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  enter edit  •  d delete  •  m meta  •  ctrl+s save  •  esc cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  "+title),
		"",
		wrapInBox(title, inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func stepLabel(sd stepDraft) string {
	if sd.stepType == "builtin" {
		return fmt.Sprintf("[builtin] %s %s", sd.action, sd.path)
	}
	run := sd.run
	if len(run) > 55 {
		run = run[:52] + "..."
	}
	return fmt.Sprintf("[cmd] %s", run)
}

func (te templateEditor) viewStepEdit(w, h int) string {
	focusIndicator := func(idx int, label string) string {
		if te.editFocus == idx {
			return promptStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}

	// Step type selector
	typeLabels := []string{"command", "builtin"}
	var typeParts []string
	for i, tl := range typeLabels {
		if i == te.editTypeCursor {
			typeParts = append(typeParts, activeStyle.Render(" "+tl+" "))
		} else {
			typeParts = append(typeParts, dimStyle.Render(tl))
		}
	}
	typeLine := focusIndicator(0, "Type: ") + strings.Join(typeParts, dimStyle.Render("  ·  ")) +
		dimStyle.Render("  (h/l or ←/→ to switch)")

	postLabel := "add to post-steps: "
	if te.editIsPost {
		postLabel += lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("yes")
	} else {
		postLabel += dimStyle.Render("no")
	}

	var inner strings.Builder
	inner.WriteString(typeLine + "\n\n")

	lastFieldIdx := 0
	if te.editTypeCursor == 0 {
		inner.WriteString(focusIndicator(1, "Command:") + "\n")
		inner.WriteString(te.editRunInput.View() + "\n\n")
		inner.WriteString(dimStyle.Render("  Tip: use {{project_name}}, {{project_path}}, {{domain}}, {{root}}") + "\n\n")
		lastFieldIdx = 2 // post-step is field index 2 (editFieldCount=3, last=2)
	} else {
		// action selector — field index 1
		var actParts []string
		for i, a := range builtinActions {
			if i == te.editActCursor {
				actParts = append(actParts, activeStyle.Render(" "+a+" "))
			} else {
				actParts = append(actParts, dimStyle.Render(a))
			}
		}
		actLine := focusIndicator(1, "Action: ") + strings.Join(actParts, dimStyle.Render("  ·  ")) +
			dimStyle.Render("  (h/l to cycle)")
		inner.WriteString(actLine + "\n\n")
		// path input — field index 2
		inner.WriteString(focusIndicator(2, "Path (relative):") + "\n")
		inner.WriteString(te.editPathInput.View() + "\n\n")
		lastFieldIdx = 3 // post-step is field index 3 (editFieldCount=4, last=3)
	}

	postFocusIdx := lastFieldIdx
	postLineStr := focusIndicator(postFocusIdx, postLabel)
	if te.editFocus == postFocusIdx {
		postLineStr += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(postLineStr + "\n")

	if te.statusMsg != "" {
		inner.WriteString("\n" + te.statusMsg)
	}
	if te.typing {
		inner.WriteString("\n\n" + hintStyle.Render("typing mode  •  esc to stop typing  •  ctrl+s save all"))
	} else {
		inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to type  •  h/l cycle type/action  •  enter confirm  •  esc back  •  ctrl+s save all"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Edit Step"),
		"",
		wrapInBox("Edit Step", inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (te templateEditor) viewMetaEdit(w, h int) string {
	focusIndicator := func(idx int, label string) string {
		if te.metaFocus == idx {
			return promptStyle.Render("▶ " + label)
		}
		return dimStyle.Render("  " + label)
	}
	boolStr := func(b bool) string {
		if b {
			return lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("yes")
		}
		return dimStyle.Render("no")
	}

	var inner strings.Builder
	inner.WriteString(focusIndicator(0, "Name:") + "\n")
	inner.WriteString(te.nameInput.View() + "\n\n")
	inner.WriteString(focusIndicator(1, "Description:") + "\n")
	inner.WriteString(te.descInput.View() + "\n\n")

	interactiveLine := focusIndicator(2, "Interactive (uses PTY): ") + boolStr(te.interactive)
	if te.metaFocus == 2 {
		interactiveLine += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(interactiveLine + "\n\n")

	folderLine := focusIndicator(3, "Creates project folder: ") + boolStr(te.createsProjectFolder)
	if te.metaFocus == 3 {
		folderLine += dimStyle.Render("  (enter/space to toggle)")
	}
	inner.WriteString(folderLine + "\n")

	if te.typing {
		inner.WriteString("\n\n" + hintStyle.Render("typing mode  •  esc to stop typing  •  ctrl+s save all"))
	} else {
		inner.WriteString("\n\n" + hintStyle.Render("j/k navigate  •  i to type  •  enter/space toggle bools  •  esc back  •  ctrl+s save all"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		promptStyle.Render("  Template Meta"),
		"",
		wrapInBox("Template Meta", inner.String()),
	)
	return centerInTerminal(w, h, content)
}

func (te templateEditor) viewConfirmDel(w, h int) string {
	label := ""
	if te.cursor < len(te.steps) {
		label = stepLabel(te.steps[te.cursor])
	}
	inner := warningStyle.Render(fmt.Sprintf("Delete step \"%s\"?", label)) + "\n\n" +
		hintStyle.Render("y/enter — yes  •  n/esc — no")
	content := lipgloss.JoinVertical(lipgloss.Left,
		RenderTitle(),
		wrapInWarningBox("Confirm Delete", inner),
	)
	return centerInTerminal(w, h, content)
}

// ===========================================================================
// Persistence helpers
// ===========================================================================
