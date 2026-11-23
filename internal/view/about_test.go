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
		SetLanguage:    func(app.Language) {},
		CommitPath:     func(string) bool { return true },
		Browse:         func() {},
		CheckAppUpdate: func() { *opened = *opened + 1 },
		OpenProject:    func() { *opened = *opened + 1 },
		OpenDownload:   func() { *opened = *opened + 1 },
		WizardRerun:    func() { *opened = *opened + 1 },
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
	tt, opened := aboutTab(t, app.Facts{AppVersion: "0.1.0"})

	if got := tt.Texts(); !slices.Contains(got, "软件版本") || !slices.Contains(got, "0.1.0") {
		t.Fatalf("texts = %q, want the running version", got)
	}
	clickButton(t, tt, "项目主页")
	if *opened != 1 {
		t.Errorf("the project page opened %d times, want once", *opened)
	}
}

func TestTheAboutTabChecksTheToolOnlyWhenClicked(t *testing.T) {
	tt, opened := aboutTab(t, app.Facts{AppVersion: "0.1.0"})
	if *opened != 0 {
		t.Fatalf("the check ran %d times before a click", *opened)
	}

	clickButton(t, tt, "检查应用更新")
	if *opened != 1 {
		t.Errorf("the check ran %d times, want once", *opened)
	}
}

func TestTheAboutTabNamesTheNewerVersionWhenOneIsKnown(t *testing.T) {
	tests := []struct {
		name  string
		facts app.Facts
		says  string
	}{
		{name: "before a check"},
		{name: "a version is available", facts: app.Facts{UpdateAvailable: "0.6.0"}, says: "发现新版本 0.6.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.facts.AppVersion = "0.5.0"
			tt, _ := aboutTab(t, test.facts)
			got := tt.Texts()
			if test.says == "" {
				for _, said := range []string{"发现新版本 0.5.0", "发现新版本 0.6.0"} {
					if slices.Contains(got, said) {
						t.Fatalf("texts = %q, no version is named before a check", got)
					}
				}
				return
			}
			if !slices.Contains(got, test.says) {
				t.Fatalf("texts = %q, want %q", got, test.says)
			}
		})
	}
}

func TestTheAboutTabAlwaysOffersTheCheckAndTheDownloadPage(t *testing.T) {
	for _, busy := range []app.Busy{app.Idle, app.BusyCheckUpdate, app.BusyApplyPatch} {
		tt, opened := aboutTab(t, app.Facts{AppVersion: "0.5.0", Busy: busy})
		check, found := tt.Access("检查应用更新")
		if !found {
			t.Fatalf("busy %v: the check is not on the tab", busy)
		}
		if check.Disabled {
			t.Errorf("busy %v: the check is disabled, want it live", busy)
		}
		clickButton(t, tt, "打开下载页")
		if *opened != 1 {
			t.Errorf("busy %v: the download page opened %d times, want once", busy, *opened)
		}
	}
}

func TestTheAboutTabRerunsTheWizard(t *testing.T) {
	tt, opened := aboutTab(t, app.Facts{AppVersion: "0.1.0"})

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
	if got := tt.Texts(); !slices.Contains(got, "沿用当前语言、主题与路径") {
		t.Errorf("texts = %q, want what the rerun starts from", got)
	}
}
