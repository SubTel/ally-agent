package desktop

import (
	backend "ally-dev/internal/app"
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestHistoryWorkspaceAlwaysCreatesNewTab(t *testing.T) {
	a := &desktop{tabs: []*conversation{{ID: "existing", Workspace: `C:\work\cam`}, {ID: "other", Workspace: `C:\other\cam`}}, active: 1}
	for _, root := range []string{`C:\work\cam`, `C:\work\cam`, `C:\other\cam`} {
		before := len(a.tabs)
		a.openWorkspaceEntry(root)
		if len(a.tabs) != before+1 || a.active != before {
			t.Fatal("history reused an existing tab")
		}
		added := a.tabs[a.active]
		if added.Workspace != root {
			t.Fatal("history opened wrong workspace")
		}
		for _, old := range a.tabs[:before] {
			if old.ID == added.ID {
				t.Fatal("history reused an existing session")
			}
		}
	}
	if a.tabs[0].ID != "existing" || a.tabs[1].ID != "other" {
		t.Fatal("existing tabs changed")
	}
}

func TestWorkspaceMenuOpensToLeftAndDistinguishesSameNames(t *testing.T) {
	a := &desktop{sessions: []backend.SessionIndexEntry{{Workspace: "C:/work/cam"}, {Workspace: "C:/other/cam"}, {Workspace: "C:/work/cam"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Absolute().Left(500).Top(10).Children(func() {
			a.workspaceHistoryMenu(c.Key("add"), func(b ui.Element) { b.Size(30, 30).Label("add") })
		})
	}, 600, 500)
	if err := tt.Click("add"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	panel, ok := tt.Find("工作区历史菜单")
	if !ok {
		t.Fatal("missing menu")
	}
	if panel.X >= 500 || panel.X+panel.W > 531 {
		t.Fatalf("menu should open left of anchor: %v", panel)
	}
	for _, root := range []string{"C:/work/cam", "C:/other/cam"} {
		if _, ok := tt.Find("打开工作区 " + root); !ok {
			t.Fatal("same-name workspace missing: " + root)
		}
	}
	if err := tt.Click("打开工作区 C:/other/cam"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("临时工作区") || len(a.tabs) != 1 || a.tabs[0].Workspace != "C:/other/cam" {
		t.Fatal("history selected wrong path or did not close")
	}
}

// The tab strip's plus opens another tab; the workspace menu belongs to the
// header's switcher alone. Two entries into one panel used to mean a press that
// switched entries closed one panel while the other went up in the same frame.
func TestTabStripPlusOpensATabAndTheSwitcherOwnsTheMenu(t *testing.T) {
	a := &desktop{
		ready:    true,
		page:     "chat",
		prefs:    defaultPrefs(),
		tabs:     []*conversation{{ID: "first", Workspace: "work"}},
		sessions: []backend.SessionIndexEntry{{Workspace: "work"}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(a.theme())
		c.DisableAnimations()
		a.headerBar(c)
	}, 900, 120)

	if err := tt.Click("新建标签"); err != nil {
		t.Fatalf("the plus beside the last tab: %v", err)
	}
	tt.Frame()
	if len(a.tabs) != 2 {
		t.Fatalf("the plus did not open a tab: %d tabs", len(a.tabs))
	}
	if a.tabs[1].Workspace != "work" {
		t.Fatalf("the plus opened %q instead of the current workspace", a.tabs[1].Workspace)
	}
	if _, ok := tt.Find("工作区历史菜单"); ok {
		t.Fatal("the plus opened the workspace menu")
	}

	if err := tt.Click("工作区：work"); err != nil {
		t.Fatalf("the header's workspace switcher: %v", err)
	}
	tt.Frame()
	count := 0
	for _, text := range tt.Texts() {
		if text == "工作区历史菜单" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("the switcher left %d menus visible: %v", count, tt.Texts())
	}
	if _, ok := tt.Find("工作区历史菜单"); !ok {
		t.Fatalf("no menu under the switcher: %v", tt.Texts())
	}
}
