package view_test

import (
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/view"

	"github.com/egoist/mygo/ui"
)

func aboutTab(t *testing.T, facts app.Facts) (*ui.Tester, *int) {
	t.Helper()
	if facts.Locale == "" {
		facts.Locale = "zh-CN"
	}
	opened := new(int)
	s := &screen{facts: facts}
	s.actions = view.Actions{
		SetLanguage:     func(app.Language) {},
		CommitPath:      func(string) bool { return true },
		Browse:          func() {},
		CheckToolUpdate: func() { *opened = *opened + 1 },
		OpenProject:     func() { *opened = *opened + 1 },
		OpenDownload:    func() { *opened = *opened + 1 },
		WizardRerun:     func() { *opened = *opened + 1 },
	}
	tt := ui.NewTester(s.view, 480, 700)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("关于"); err != nil {
		t.Fatalf("click the About tab: %v", err)
	}
	return tt, opened
}

func TestTheAboutTabShowsTheVersionAndTheProjectPage(t *testing.T) {
	tt, opened := aboutTab(t, app.Facts{ToolVersion: "0.1.0"})

	if got := tt.Texts(); !slices.Contains(got, "软件版本") || !slices.Contains(got, "0.1.0") {
		t.Fatalf("texts = %q, want the running version", got)
	}
	clickButton(t, tt, "项目主页")
	if *opened != 1 {
		t.Errorf("the project page opened %d times, want once", *opened)
	}
}

func TestTheAboutTabChecksTheToolOnlyWhenClicked(t *testing.T) {
	tt, opened := aboutTab(t, app.Facts{ToolVersion: "0.1.0"})
	if *opened != 0 {
		t.Fatalf("the check ran %d times before a click", *opened)
	}

	clickButton(t, tt, "检查应用更新")
	if *opened != 1 {
		t.Errorf("the check ran %d times, want once", *opened)
	}
}

func TestTheAboutTabReportsTheToolCheckResult(t *testing.T) {
	tests := []struct {
		name  string
		facts app.Facts
		says  string
	}{
		{name: "before a check"},
		{name: "while checking", facts: app.Facts{Busy: app.BusyCheckTool}, says: "正在检查应用更新..."},
		{name: "up to date", facts: app.Facts{ToolChecked: true, ToolOnline: "0.1.0"}, says: "已是最新版本"},
		{name: "newer", facts: app.Facts{ToolChecked: true, ToolOnline: "0.2.0"}, says: "发现新版本 0.2.0"},
		{name: "failed", facts: app.Facts{ToolFailed: true}, says: "检查失败"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.facts.ToolVersion = "0.1.0"
			tt, opened := aboutTab(t, test.facts)
			got := tt.Texts()
			if test.says == "" {
				for _, said := range []string{"正在检查应用更新...", "检查失败", "已是最新版本"} {
					if slices.Contains(got, said) {
						t.Fatalf("texts = %q, before a check nothing is said", got)
					}
				}
			} else if !slices.Contains(got, test.says) {
				t.Fatalf("texts = %q, want %q", got, test.says)
			}
			check, found := tt.Access("检查应用更新")
			if !found {
				t.Fatal("the check is not on the tab")
			}
			if check.Disabled != (test.facts.Busy == app.BusyCheckTool) {
				t.Errorf("the check is disabled %v, want %v", check.Disabled, test.facts.Busy == app.BusyCheckTool)
			}
			if test.name == "newer" {
				clickButton(t, tt, "打开下载页")
				if *opened != 1 {
					t.Errorf("the download page opened %d times, want once", *opened)
				}
			} else if _, found := tt.Find("打开下载页"); found {
				t.Error("a download page is offered when there is nothing newer")
			}
		})
	}
}

func TestTheAboutTabRerunsTheWizard(t *testing.T) {
	tt, opened := aboutTab(t, app.Facts{ToolVersion: "0.1.0"})

	rerun, found := tt.Access("重新运行向导")
	if !found {
		t.Fatal("the wizard rerun is not on the tab")
	}
	if rerun.Disabled {
		t.Error("the wizard rerun is held while nothing blocks it")
	}
	clickButton(t, tt, "重新运行向导")
	if *opened != 1 {
		t.Errorf("the rerun ran %d times, want once", *opened)
	}
	if got := tt.Texts(); !slices.Contains(got, "以当前语言、主题与 Steam 路径为初值重新运行设置向导。") {
		t.Errorf("texts = %q, want what the rerun starts from", got)
	}
}

func TestTheAboutTabWaitsWhileAnotherCommandRuns(t *testing.T) {
	tt, _ := aboutTab(t, app.Facts{ToolVersion: "0.1.0", Busy: app.BusyCheckUpdate})

	check, found := tt.Access("检查应用更新")
	if !found {
		t.Fatal("the check is not on the tab")
	}
	if !check.Disabled {
		t.Error("the check is live while another command holds the gate")
	}
}
