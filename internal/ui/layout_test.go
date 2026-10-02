package ui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestLayoutBoundariesAndResize(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 15}, {24, 8}, {23, 8}, {24, 7}, {0, 0}, {-1, -1}, {300, 80}} {
		l := Measure(size[0], size[1])
		if l.Mode == Wide && l.Workspace+l.Projects+l.Inspector+2*PaneGap != l.Width {
			t.Fatalf("wide allocation %+v", l)
		}
		if l.Mode == Medium && l.Workspace+l.Projects+PaneGap != l.Width {
			t.Fatalf("medium allocation %+v", l)
		}
		if l.Mode != Tiny && l.BodyHeight+ChromeHeight != l.Height {
			t.Fatalf("height %+v", l)
		}
		if l.Mode == Wide && (l.Workspace < WorkspaceMin || l.Projects < ProjectsMin || l.Inspector < InspectorMin) {
			t.Fatalf("crushed pane %+v", l)
		}
	}
	if Measure(97, 24).Mode != Medium || Measure(98, 24).Mode != Wide || Measure(65, 24).Mode != Narrow || Measure(66, 24).Mode != Medium {
		t.Fatal("breakpoint boundary")
	}
}

func TestCellAwareComponents(t *testing.T) {
	for _, size := range [][2]int{{40, 15}, {24, 8}, {1, 1}, {0, 0}} {
		text := "👩🏽‍💻 café e\u0301 日本語 " + strings.Repeat("長", 100)
		for _, content := range []string{Fit(text, size[0], size[1]), Modal(text, text, "esc close", size[0], size[1]), TooSmall(size[0], size[1])} {
			lines := strings.Split(content, "\n")
			if content != "" && len(lines) > size[1] {
				t.Fatalf("height %v", size)
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("width %v: %q", size, line)
				}
			}
		}
	}
}

func TestFocusCycle(t *testing.T) {
	if Projects.Cycle(1, false) != Workspace || Workspace.Cycle(-1, false) != Projects {
		t.Fatal("foundation focus")
	}
	if Projects.Cycle(1, true) != Inspector || Inspector.Cycle(1, true) != Workspace || Workspace.Cycle(-1, true) != Inspector {
		t.Fatal("dashboard focus")
	}
}
