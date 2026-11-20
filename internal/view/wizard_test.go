package view_test

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/view"
)

func wizardScreen(t *testing.T, facts app.Facts, actions view.Actions) (*screen, *ui.Tester) {
	t.Helper()
	if facts.Locale == "" {
		facts.Locale = "zh-CN"
	}
	s := &screen{facts: facts, actions: actions}
	return s, ui.NewTester(s.view, 480, 360)
}

func openWizard(facts app.Facts) app.Facts {
	facts.Wizard.Open = true
	return facts
}

func TestTheWizardReplacesTheMainScreenWithItsStep(t *testing.T) {
	for _, step := range []struct {
		step app.WizardStep
		text string
	}{
		{app.WizardLanguage, "界面语言"},
		{app.WizardTheme, "主题"},
		{app.WizardPath, "Steam 路径"},
		{app.WizardDownload, "第 4 / 4 步"},
	} {
		facts := openWizard(app.Facts{SteamPathValid: true, PatchDownloaded: true})
		facts.Wizard.Step = step.step
		_, tt := wizardScreen(t, facts, view.Actions{})
		if !tt.HasText("首次运行向导") {
			t.Fatalf("step %v: texts %q, want the wizard title", step.step, tt.Texts())
		}
		if !tt.HasText(step.text) {
			t.Fatalf("step %v: texts %q, want %q", step.step, tt.Texts(), step.text)
		}
		if tt.HasText("检查补丁更新") || tt.HasText("Steam 未运行") {
			t.Errorf("step %v: texts %q, the main screen shows behind the wizard", step.step, tt.Texts())
		}
	}
}

func TestEveryWizardStepNamesItselfAndSaysWhatItChanges(t *testing.T) {
	tests := []struct {
		step app.WizardStep
		name string
		hint string
	}{
		{app.WizardLanguage, "界面语言", "影响窗口、对话框和托盘菜单里的全部文字。"},
		{app.WizardTheme, "主题", "影响整个窗口的配色。"},
		{app.WizardPath, "Steam 路径", "补丁会部署到这个 Steam 安装目录。"},
		{app.WizardDownload, "下载补丁", "获取与当前版本匹配的补丁文件。"},
	}
	for _, test := range tests {
		facts := openWizard(app.Facts{SteamPathValid: true, PatchDownloaded: true})
		facts.Wizard.Step = test.step
		_, tt := wizardScreen(t, facts, view.Actions{})
		if !tt.HasText(test.name) {
			t.Errorf("step %v: texts %q, want the step name %q", test.step, tt.Texts(), test.name)
		}
		if !tt.HasText(test.hint) {
			t.Errorf("step %v: texts %q, want the hint %q", test.step, tt.Texts(), test.hint)
		}
	}
}

func TestTheLanguageStepPreviewsTheChosenLanguage(t *testing.T) {
	for _, test := range []struct {
		locale string
		want   string
		absent string
	}{
		{"zh-CN", "应用补丁并启动 Steam", "Apply patch and start Steam"},
		{"en-US", "Apply patch and start Steam", "应用补丁并启动 Steam"},
	} {
		facts := openWizard(app.Facts{Locale: test.locale})
		_, tt := wizardScreen(t, facts, view.Actions{})
		if sample, found := tt.Find(test.want); !found || sample.W == 0 {
			t.Errorf("%s: sample %+v found %v, texts %q, want the sample %q on screen", test.locale, sample, found, tt.Texts(), test.want)
		}
		if tt.HasText(test.absent) {
			t.Errorf("%s: texts %q, the sample stays in the other language", test.locale, tt.Texts())
		}
	}
}

func TestTheThemeStepShowsBothAppearances(t *testing.T) {
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardTheme
	_, tt := wizardScreen(t, facts, view.Actions{})
	for _, want := range []string{"浅色", "深色"} {
		if card, found := tt.Find(want); !found || card.W == 0 {
			t.Errorf("card %+v found %v, texts %q, want the %s appearance on screen", card, found, tt.Texts(), want)
		}
	}
}

func TestTheWizardBlocksNextWithoutAValidPath(t *testing.T) {
	next := 0
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardPath
	_, tt := wizardScreen(t, facts, view.Actions{WizardNext: func() { next++ }})

	button, found := tt.Access("下一步")
	if !found {
		t.Fatalf("texts %q, want the next control", tt.Texts())
	}
	if !button.Disabled {
		t.Error("next is live with an empty path")
	}

	facts.SteamPathValid = true
	_, tt = wizardScreen(t, facts, view.Actions{WizardNext: func() { next++ }})
	if button, found := tt.Access("下一步"); !found || button.Disabled {
		t.Fatalf("next = %+v found %v, want it live with a valid path", button, found)
	}
	if err := tt.Click("下一步"); err != nil {
		t.Fatalf("click next: %v", err)
	}
	if next != 1 {
		t.Errorf("next ran %d times, want once", next)
	}
}

func TestTheWizardPicksALanguageFromADropdown(t *testing.T) {
	chosen := []app.Language{}
	facts := openWizard(app.Facts{})
	_, tt := wizardScreen(t, facts, view.Actions{WizardLanguage: func(l app.Language) { chosen = append(chosen, l) }})

	if err := tt.Click("跟随系统"); err != nil {
		t.Fatalf("open the language dropdown: %v", err)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatalf("choose English: %v", err)
	}
	if !slices.Equal(chosen, []app.Language{app.LanguageEnglish}) {
		t.Fatalf("chosen = %v, want the English preference", chosen)
	}
}

func TestTheWizardPicksAThemeFromADropdown(t *testing.T) {
	picked := []app.Theme{}
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardTheme
	_, tt := wizardScreen(t, facts, view.Actions{WizardTheme: func(theme app.Theme) { picked = append(picked, theme) }})

	if err := tt.Click("跟随系统"); err != nil {
		t.Fatalf("open the theme dropdown: %v", err)
	}
	if err := tt.Click("浅色"); err != nil {
		t.Fatalf("choose the light theme: %v", err)
	}
	if !slices.Equal(picked, []app.Theme{app.ThemeLight}) {
		t.Fatalf("picked = %v, want the light theme", picked)
	}
}

func TestTheWizardWalksBackThroughItsSteps(t *testing.T) {
	back := 0
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardTheme
	_, tt := wizardScreen(t, facts, view.Actions{WizardBack: func() { back++ }})

	if err := tt.Click("上一步"); err != nil {
		t.Fatalf("click back: %v", err)
	}
	if back != 1 {
		t.Fatalf("back ran %d times, want once", back)
	}
	facts.Wizard.Step = app.WizardLanguage
	_, tt = wizardScreen(t, facts, view.Actions{})
	if _, found := tt.Access("上一步"); found {
		t.Error("the first step offers a way back")
	}
}

func TestThePathStepDraftsThePatchSource(t *testing.T) {
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardPath
	_, tt := wizardScreen(t, facts, view.Actions{})
	if !tt.HasText("社区维护") || !tt.HasText("自定义仓库") {
		t.Fatalf("texts %q, want the patch source draft", tt.Texts())
	}
}

func TestTheWizardDownloadStepTakesItsShapeFromTheLocalFiles(t *testing.T) {
	tests := []struct {
		name    string
		shape   app.WizardDownloadState
		failure app.Result
		texts   []string
		buttons []string
		absent  []string
	}{
		{
			name:    "not downloaded",
			shape:   app.WizardNotDownloaded,
			texts:   []string{"补丁尚未下载。点击下方按钮开始下载并解压（可选跳过）。"},
			buttons: []string{"下载并解压", "跳过"},
			absent:  []string{"完成", "重试"},
		},
		{
			name:    "already downloaded",
			shape:   app.WizardDownloaded,
			texts:   []string{"补丁已下载，无需再下载。"},
			buttons: []string{"完成"},
			absent:  []string{"下载并解压", "跳过", "重试"},
		},
		{
			name:    "downloading",
			shape:   app.WizardDownloading,
			texts:   []string{"正在下载并解压补丁..."},
			buttons: []string{"跳过"},
			absent:  []string{"完成", "重试", "下载并解压"},
		},
		{
			name:    "failed",
			shape:   app.WizardDownloadFailed,
			failure: app.NetworkFailed,
			texts:   []string{"补丁下载失败：网络请求失败"},
			buttons: []string{"重试", "跳过"},
			absent:  []string{"完成", "下载并解压"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts := openWizard(app.Facts{})
			facts.Wizard.Step = app.WizardDownload
			facts.Wizard.Download = test.shape
			facts.Wizard.Failure = test.failure
			_, tt := wizardScreen(t, facts, view.Actions{})

			for _, want := range test.texts {
				if !tt.HasText(want) {
					t.Errorf("texts %q, want %q", tt.Texts(), want)
				}
			}
			for _, want := range test.buttons {
				if _, found := tt.Access(want); !found {
					t.Errorf("texts %q, want the %s button", tt.Texts(), want)
				}
			}
			for _, unwanted := range test.absent {
				if _, found := tt.Access(unwanted); found {
					t.Errorf("texts %q, %q does not belong to this shape", tt.Texts(), unwanted)
				}
			}
		})
	}
}

func TestTheWizardDownloadRunsFromItsOwnButtons(t *testing.T) {
	calls := []string{}
	actions := view.Actions{
		WizardDownload: func() { calls = append(calls, "download") },
		WizardSkip:     func() { calls = append(calls, "skip") },
		WizardFinish:   func() { calls = append(calls, "finish") },
	}
	facts := openWizard(app.Facts{})
	facts.Wizard.Step = app.WizardDownload
	facts.Wizard.Download = app.WizardDownloadFailed
	facts.Wizard.Failure = app.ParseFailed
	_, tt := wizardScreen(t, facts, actions)
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("click retry: %v", err)
	}
	if err := tt.Click("跳过"); err != nil {
		t.Fatalf("click skip: %v", err)
	}

	facts.Wizard.Download = app.WizardDownloaded
	_, tt = wizardScreen(t, facts, actions)
	if err := tt.Click("完成"); err != nil {
		t.Fatalf("click finish: %v", err)
	}
	if !slices.Equal(calls, []string{"download", "skip", "finish"}) {
		t.Errorf("calls = %v, want the download, the skip and the finish", calls)
	}
}

func overlaps(a, b ui.Rect) bool {
	if a.W <= 0 || a.H <= 0 || b.W <= 0 || b.H <= 0 {
		return false
	}
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestTheWizardFooterNeverOverlapsTheStepBody(t *testing.T) {
	footers := map[string]bool{"上一步": true, "下一步": true, "跳过": true, "完成": true, "重试": true, "下载并解压": true}
	for _, size := range []struct{ w, h int }{{480, 360}, {320, 280}} {
		for _, step := range []app.WizardStep{app.WizardLanguage, app.WizardTheme, app.WizardPath, app.WizardDownload} {
			for _, shape := range []app.WizardDownloadState{app.WizardNotDownloaded, app.WizardDownloading, app.WizardDownloadFailed, app.WizardDownloaded} {
				facts := openWizard(app.Facts{SteamPath: "C:/Steam", SteamPathValid: true})
				facts.Wizard.Step = step
				facts.Wizard.Download = shape
				facts.Wizard.Failure = app.NetworkFailed
				_, tt := wizardScreen(t, facts, view.Actions{})
				tt.SetSize(size.w, size.h)

				texts := tt.Texts()
				for _, body := range texts {
					if footers[body] {
						continue
					}
					bodyRect, ok := tt.Find(body)
					if !ok {
						continue
					}
					for _, footer := range texts {
						if !footers[footer] {
							continue
						}
						footerRect, ok := tt.Find(footer)
						if !ok {
							continue
						}
						if overlaps(bodyRect, footerRect) {
							t.Errorf("%dx%d step %d shape %d: %q at %+v overlaps the footer %q at %+v", size.w, size.h, step, shape, body, bodyRect, footer, footerRect)
						}
					}
				}
			}
		}
	}
}

func TestTheWizardControlsStayApart(t *testing.T) {
	facts := openWizard(app.Facts{SteamPathValid: true})
	facts.Wizard.Step = app.WizardDownload
	facts.Wizard.Download = app.WizardNotDownloaded
	_, tt := wizardScreen(t, facts, view.Actions{})
	back, _ := tt.Find("上一步")
	skip, _ := tt.Find("跳过")
	primary, _ := tt.Find("下载并解压")
	for _, pair := range []struct {
		name string
		a, b ui.Rect
	}{{"back and skip", back, skip}, {"skip and primary", skip, primary}, {"back and primary", back, primary}} {
		if overlaps(pair.a, pair.b) {
			t.Errorf("%s overlap: %+v and %+v", pair.name, pair.a, pair.b)
		}
	}
	if !(back.X < skip.X && skip.X < primary.X) {
		t.Errorf("controls sit at %+v, %+v, %+v, want back, skip, then the primary", back, skip, primary)
	}
}

func TestThePathStepScrollsToItsSourceChoices(t *testing.T) {
	facts := openWizard(app.Facts{SteamPath: "C:/Steam", SteamPathValid: true})
	facts.Wizard.Step = app.WizardPath
	_, tt := wizardScreen(t, facts, view.Actions{})

	last, found := tt.Find("OpenSteamTool 布局")
	if !found {
		t.Fatalf("texts %q, want the layout conclusion", tt.Texts())
	}
	if last.W != 0 {
		t.Fatalf("the layout conclusion at %+v already fits, nothing to scroll", last)
	}

	tt.Move(240, 150)
	tt.Scroll(240, 150, 0, 400)
	last, found = tt.Find("OpenSteamTool 布局")
	if !found || last.W == 0 {
		t.Fatalf("the layout conclusion at %+v stays hidden after scrolling", last)
	}
}

func TestTheLanguageSampleStaysReachableAtTheSmallestWindow(t *testing.T) {
	facts := openWizard(app.Facts{Locale: "zh-CN"})
	s := &screen{facts: facts}
	tt := ui.NewTester(s.view, 320, 280)

	sample, found := tt.Find("应用补丁并启动 Steam")
	if !found {
		t.Fatalf("texts %q, want the language sample", tt.Texts())
	}
	if sample.W != 0 {
		t.Fatalf("the sample at %+v already fits, nothing to scroll", sample)
	}

	tt.Move(160, 140)
	tt.Scroll(160, 140, 0, 200)
	sample, found = tt.Find("应用补丁并启动 Steam")
	if !found || sample.W == 0 {
		t.Fatalf("the sample at %+v stays hidden after scrolling", sample)
	}
}

func TestTheWizardStaysClickableAtTheSmallestWindow(t *testing.T) {
	next, back := 0, 0
	facts := openWizard(app.Facts{SteamPath: "C:/Steam", SteamPathValid: true})
	facts.Wizard.Step = app.WizardTheme
	_, tt := wizardScreen(t, facts, view.Actions{
		WizardNext: func() { next++ },
		WizardBack: func() { back++ },
	})
	tt.SetSize(320, 280)

	if primary, found := tt.Access("下一步"); !found || primary.Disabled {
		t.Fatalf("next = %+v found %v, want it live", primary, found)
	}
	rect, found := tt.Find("下一步")
	if !found {
		t.Fatalf("texts %q, want the next control", tt.Texts())
	}
	if rect.Y+rect.H > 280 {
		t.Errorf("the next control at %+v sits outside the window", rect)
	}
	if err := tt.Click("下一步"); err != nil {
		t.Fatalf("click next: %v", err)
	}
	if back := tt.Click("上一步"); back != nil {
		t.Fatalf("click back: %v", back)
	}
	if next != 1 || back != 1 {
		t.Errorf("next ran %d times and back %d, want once each", next, back)
	}
}
