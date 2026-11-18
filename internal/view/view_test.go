package view_test

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/view"
)

type screen struct {
	facts   app.Facts
	state   view.State
	actions view.Actions
}

func (s *screen) view(c *ui.Context) {
	view.Main(c, app.Derive(s.facts), &s.state, s.actions)
}

func shown(t *testing.T, facts app.Facts) *ui.Tester {
	t.Helper()
	if facts.Locale == "" {
		facts.Locale = "zh-CN"
	}
	s := &screen{facts: facts}
	return ui.NewTester(s.view, 480, 360)
}

func TestMainDerivesOneStateAndAtMostOneExplanation(t *testing.T) {
	invalid := []string{"Steam 路径无效", "请在设置中指定有效的 Steam 安装路径"}
	notDownloaded := []string{"补丁未下载", "请点击「检查补丁更新」下载新版本"}
	with := func(word string, rest ...string) []string {
		return append([]string{word, "Steam 未运行", "应用补丁并启动 Steam", "检查补丁更新", "设置"}, rest...)
	}
	busyWith := func(stage string) []string {
		return []string{stage, stage, "Steam 未运行", "应用补丁并启动 Steam", "检查补丁更新", "设置"}
	}
	tests := []struct {
		name  string
		facts app.Facts
		texts []string
	}{
		{
			name:  "applied",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
			texts: with("已应用"),
		},
		{
			name:  "not applied",
			facts: app.Facts{SteamPathValid: true, Deployment: app.NotApplied, PatchDownloaded: true},
			texts: with("未应用"),
		},
		{
			name:  "steam running",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, SteamRunning: true},
			texts: []string{"已应用", "Steam 正在运行", "应用补丁并启动 Steam", "检查补丁更新", "设置"},
		},
		{
			name:  "the last result follows the steam slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, SteamRunning: true, Result: app.SteamRestarted},
			texts: []string{"已应用", "Steam 正在运行", "Steam 已重启", "应用补丁并启动 Steam", "检查补丁更新", "设置"},
		},
		{
			name:  "a failed close lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.CloseSteamFailed},
			texts: with("已应用", "关闭 Steam 失败"),
		},
		{
			name:  "a missing steam.exe lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.MissingSteamProgram},
			texts: with("已应用", "未找到 steam.exe"),
		},
		{
			name:  "an applied patch lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.PatchApplied},
			texts: with("已应用", "补丁已应用"),
		},
		{
			name:  "a refused apply without a path lands in the result slot",
			facts: app.Facts{SteamPathValid: false, Result: app.PathUnset},
			texts: append(with("未应用", "请先指定有效的 Steam 安装路径"), invalid...),
		},
		{
			name:  "a failed deploy lands in the result slot",
			facts: app.Facts{SteamPathValid: true, PatchDownloaded: true, Result: app.DeploymentFailed},
			texts: with("未应用", "部署失败"),
		},
		{
			name:  "invalid path",
			facts: app.Facts{SteamPathValid: false, Deployment: app.NotApplied, PatchDownloaded: true},
			texts: with("未应用", invalid...),
		},
		{
			name:  "invalid path keeps the applied fact out of the word",
			facts: app.Facts{SteamPathValid: false, Deployment: app.Applied, PatchDownloaded: true},
			texts: with("未应用", invalid...),
		},
		{
			name:  "patch not downloaded",
			facts: app.Facts{SteamPathValid: true, Deployment: app.NotApplied, PatchDownloaded: false},
			texts: with("未应用", notDownloaded...),
		},
		{
			name:  "invalid path outranks not downloaded",
			facts: app.Facts{SteamPathValid: false, Deployment: app.NotApplied, PatchDownloaded: false},
			texts: with("未应用", invalid...),
		},
		{
			name:  "apply busy shows its stage",
			facts: app.Facts{SteamPathValid: false, Busy: app.BusyApplyPatch},
			texts: busyWith("正在应用补丁…"),
		},
		{
			name:  "uninstall busy shows its stage",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyUninstall},
			texts: busyWith("正在卸载补丁…"),
		},
		{
			name:  "start busy shows its stage",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyStartSteam},
			texts: busyWith("正在启动 Steam…"),
		},
		{
			name:  "stop busy shows its stage",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyStopSteam},
			texts: busyWith("正在关闭 Steam…"),
		},
		{
			name:  "update busy keeps the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.NotApplied, Busy: app.BusyCheckUpdate},
			texts: []string{"未应用", "Steam 未运行", "检查中…", "应用补丁并启动 Steam", "检查补丁更新", "设置"},
		},
		{
			name:  "download busy keeps the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyDownloadPatch},
			texts: []string{"已应用", "Steam 未运行", "应用补丁并启动 Steam", "检查补丁更新", "正在下载补丁…", "设置"},
		},
		{
			name:  "a failed apply outranks the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, PatchFailed: true},
			texts: with("补丁应用失败"),
		},
		{
			name:  "a failed apply keeps the path explanation",
			facts: app.Facts{SteamPathValid: false, PatchFailed: true},
			texts: with("补丁应用失败", invalid...),
		},
		{
			name:  "a failed apply keeps the download prompt while the patch is missing",
			facts: app.Facts{SteamPathValid: true, PatchFailed: true},
			texts: with("补丁应用失败", notDownloaded...),
		},
		{
			name:  "busy outranks a failed apply",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchFailed: true, Busy: app.BusyApplyPatch},
			texts: busyWith("正在应用补丁…"),
		},
		{
			name:  "english applied",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
			texts: []string{"Applied", "Steam is not running", "Settings", "Apply patch and start Steam", "Check for patch updates"},
		},
		{
			name:  "english result",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.StartSteamFailed},
			texts: []string{"Applied", "Steam is not running", "Failed to start Steam", "Settings", "Apply patch and start Steam", "Check for patch updates"},
		},
		{
			name:  "english invalid path",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: false},
			texts: []string{"Not applied", "Steam is not running", "Settings", "Steam path is invalid", "Apply patch and start Steam", "Check for patch updates", "Set a valid Steam installation path in settings"},
		},
		{
			name:  "english patch failed",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true, PatchDownloaded: true, PatchFailed: true},
			texts: []string{"Patch failed to apply", "Steam is not running", "Settings", "Apply patch and start Steam", "Check for patch updates"},
		},
		{
			name:  "english patch not downloaded",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true},
			texts: []string{"Not applied", "Steam is not running", "Settings", "Apply patch and start Steam", "Check for patch updates", "Patch not downloaded", "Click Check for patch updates to download a new version"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := shown(t, test.facts).Texts()
			slices.Sort(got)
			want := slices.Clone(test.texts)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("texts %q, want %q", got, want)
			}
		})
	}
}

func TestTheWindowCarriesTheSteamSlotWhateverHappens(t *testing.T) {
	snapshots := []app.Facts{
		{},
		{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
		{SteamPathValid: false, PatchFailed: true, Busy: app.BusyApplyPatch},
		{SteamPathValid: true, Busy: app.BusyDownloadPatch, SteamRunning: true},
	}
	for _, facts := range snapshots {
		texts := shown(t, facts).Texts()
		if !slices.Contains(texts, "Steam 未运行") && !slices.Contains(texts, "Steam 正在运行") {
			t.Errorf("no Steam slot in %q", texts)
		}
	}
}

func TestLanguageSwitchRebuildsTheSameWindow(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true}}
	tt := ui.NewTester(s.view, 480, 360)
	if !tt.HasText("已应用") {
		t.Fatalf("texts %q before the switch", tt.Texts())
	}
	s.facts.Language = app.LanguageEnglish
	tt.Frame()
	if !tt.HasText("Applied") || tt.HasText("已应用") {
		t.Errorf("texts %q after the switch", tt.Texts())
	}
}

func TestTheStateWordIsReadAsStatusText(t *testing.T) {
	tt := shown(t, app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true})
	info, ok := tt.AccessByRole(ui.RoleStatus, "已应用")
	if !ok {
		t.Fatalf("no status region named 已应用 in %+v", tt.AccessAll())
	}
	if info.Role != ui.RoleStatus || info.Name != "已应用" {
		t.Errorf("role %v name %q", info.Role, info.Name)
	}
}

func TestTheTrayItemSpeaksBothLanguages(t *testing.T) {
	if got := view.RestartSteam(app.Chinese); got != "重启 Steam" {
		t.Errorf("the Chinese tray item is %q", got)
	}
	if got := view.RestartSteam(app.English); got != "Restart Steam" {
		t.Errorf("the English tray item is %q", got)
	}
}

func dialog(t *testing.T) (*screen, *ui.Tester) {
	t.Helper()
	s := &screen{facts: app.Facts{Locale: "zh-CN"}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
	}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	return s, tt
}

func TestTheGearOpensTheSettingsDialogWithTheFiveTabs(t *testing.T) {
	s, tt := dialog(t)
	if !s.state.SettingsOpen {
		t.Error("the dialog did not open")
	}
	for _, label := range []string{"通用", "补丁源", "Steam 设置", "Lua 配置", "关于", "关闭"} {
		if !tt.HasText(label) {
			t.Errorf("the dialog shows %q without %q", tt.Texts(), label)
		}
	}
}

func TestTheCloseButtonShutsTheDialog(t *testing.T) {
	s, tt := dialog(t)
	if err := tt.Click("关闭"); err != nil {
		t.Fatalf("click close: %v", err)
	}
	if s.state.SettingsOpen {
		t.Error("the dialog stayed open")
	}
	if tt.HasText("补丁源") {
		t.Errorf("the dialog is still built: %q", tt.Texts())
	}
}

func TestAClickOutsideTheDialogKeepsItOpen(t *testing.T) {
	s, tt := dialog(t)
	tt.ClickAt(2, 2)
	if !s.state.SettingsOpen || !tt.HasText("关闭") {
		t.Errorf("the dialog closed on a click outside: %q", tt.Texts())
	}
}

func TestTheLanguageChoiceIsReportedAsSoonAsItIsPicked(t *testing.T) {
	s, tt := dialog(t)
	var picked []app.Language
	s.actions.SetLanguage = func(language app.Language) { picked = append(picked, language) }

	if err := tt.Click("英语"); err != nil {
		t.Fatalf("click the English segment: %v", err)
	}
	if !slices.Equal(picked, []app.Language{app.LanguageEnglish}) {
		t.Errorf("picked %v, want the English preference", picked)
	}
	if err := tt.Click("简体中文"); err != nil {
		t.Fatalf("click the Chinese segment: %v", err)
	}
	if !slices.Equal(picked, []app.Language{app.LanguageEnglish, app.LanguageChinese}) {
		t.Errorf("picked %v, want the Chinese preference second", picked)
	}
}

func TestTheSteamTabCommitsThePathOnEnter(t *testing.T) {
	s, tt := dialog(t)
	var committed []string
	s.actions.CommitPath = func(raw string) bool {
		committed = append(committed, raw)
		return true
	}

	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	if err := tt.Click("Steam 安装路径"); err != nil {
		t.Fatalf("focus the path field: %v", err)
	}
	tt.Type(`D:\Steam`)
	tt.Key(0, ui.KeyEnter)

	if !slices.Equal(committed, []string{`D:\Steam`}) {
		t.Errorf("committed %v, want the typed path once", committed)
	}
	if s.state.PathError {
		t.Error("an accepted path left the inline error on")
	}
}

func TestARejectedPathShowsTheInlineError(t *testing.T) {
	s, tt := dialog(t)
	s.actions.CommitPath = func(string) bool { return false }

	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Steam 安装路径"); err != nil {
		t.Fatal(err)
	}
	tt.Type(`D:\Nope`)
	tt.Key(0, ui.KeyEnter)

	if !s.state.PathError {
		t.Error("a rejected path left the field without an error")
	}
	if !tt.HasText("目录不存在或无法读取") {
		t.Errorf("texts %q, want the inline error", tt.Texts())
	}
}

func TestTheSteamTabCommitsWhenTheFieldLosesFocus(t *testing.T) {
	s, tt := dialog(t)
	var committed []string
	s.actions.CommitPath = func(raw string) bool {
		committed = append(committed, raw)
		return true
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Steam 安装路径"); err != nil {
		t.Fatal(err)
	}
	tt.Type(`D:\Steam`)
	if err := tt.Click("浏览…"); err != nil {
		t.Fatalf("click browse: %v", err)
	}

	if !slices.Contains(committed, `D:\Steam`) {
		t.Errorf("committed %v, the draft was lost when the field lost focus", committed)
	}
}

func TestTheSteamTabCommitsBeforeTheTabChanges(t *testing.T) {
	s, tt := dialog(t)
	var committed []string
	s.actions.CommitPath = func(raw string) bool {
		committed = append(committed, raw)
		return true
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Steam 安装路径"); err != nil {
		t.Fatal(err)
	}
	tt.Type(`D:\Steam`)
	if err := tt.Click("通用"); err != nil {
		t.Fatalf("switch to the General tab: %v", err)
	}

	if s.state.Tab != int(app.TabGeneral) {
		t.Fatalf("the tab is %d, want General", s.state.Tab)
	}
	if !slices.Contains(committed, `D:\Steam`) {
		t.Errorf("committed %v, the draft was lost when the tab changed", committed)
	}
}

func TestTheSteamTabKeepsTheTypedDraftWhileTheDialogMovesBetweenTabs(t *testing.T) {
	s, tt := dialog(t)
	s.actions.CommitPath = func(string) bool { return false }
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Steam 安装路径"); err != nil {
		t.Fatal(err)
	}
	tt.Type(`D:\Steam`)
	if err := tt.Click("关于"); err != nil {
		t.Fatal(err)
	}
	if s.state.PathDraft != `D:\Steam` {
		t.Errorf("the draft is %q, want the typed path", s.state.PathDraft)
	}
}

func TestBrowseAsksTheAppForADirectory(t *testing.T) {
	s, tt := dialog(t)
	asked := 0
	s.actions.Browse = func() { asked++ }

	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("浏览…"); err != nil {
		t.Fatalf("click browse: %v", err)
	}
	if asked != 1 {
		t.Errorf("browse was asked %d times, want once", asked)
	}
}

func TestTheDialogKeepsTheTabItWasLeftOn(t *testing.T) {
	s, tt := dialog(t)
	if err := tt.Click("Lua 配置"); err != nil {
		t.Fatal(err)
	}
	if s.state.Tab != int(app.TabLua) {
		t.Fatalf("the tab is %d, want the Lua tab", s.state.Tab)
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if s.state.Tab != int(app.TabLua) {
		t.Errorf("the dialog reopened on tab %d, want the remembered Lua tab", s.state.Tab)
	}
}

func TestTheSetupScreenOffersSettingsOnTheSteamTab(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", NeedsSetup: true}}
	tt := ui.NewTester(s.view, 480, 360)
	if !tt.HasText("尚未完成设置") {
		t.Fatalf("texts %q, want the setup notice", tt.Texts())
	}
	if tt.HasText("未应用") {
		t.Errorf("texts %q, the setup screen shows the main state", tt.Texts())
	}
	if err := tt.Click("打开设置"); err != nil {
		t.Fatalf("click open settings: %v", err)
	}
	if !s.state.SettingsOpen || s.state.Tab != int(app.TabSteam) {
		t.Errorf("settings open = %v on tab %d, want the Steam tab", s.state.SettingsOpen, s.state.Tab)
	}
}

func TestTheCloseButtonStaysVisibleWhileTheContentScrolls(t *testing.T) {
	s, tt := dialog(t)
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatal(err)
	}
	tt.Wheel(240, 200, 0, -400)
	if !tt.HasText("关闭") {
		t.Fatalf("texts %q, the close button scrolled out of the dialog", tt.Texts())
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatalf("click close after scrolling: %v", err)
	}
	if s.state.SettingsOpen {
		t.Error("the dialog stayed open")
	}
}

func TestTheDialogFitsTheSmallestWindowTheAppAllows(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN"}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
	}
	tt := ui.NewTester(s.view, 320, 280)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatalf("the close button is out of reach at the smallest window: %v", err)
	}
	if s.state.SettingsOpen {
		t.Error("the dialog stayed open")
	}
}

func TestTheEscapeKeyDoesNotCloseTheDialog(t *testing.T) {
	s, tt := dialog(t)
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !s.state.SettingsOpen {
		t.Error("Escape closed the settings dialog")
	}
	if !tt.HasText("关闭") {
		t.Fatalf("texts %q, the dialog is gone", tt.Texts())
	}
}

func TestTheUpdateSlotShowsOneConclusionAndTheDownloadWhenItIsOffered(t *testing.T) {
	tests := []struct {
		name     string
		facts    app.Facts
		text     string
		download bool
	}{
		{name: "before a check", facts: app.Facts{SteamPathValid: true}},
		{name: "up to date", facts: app.Facts{SteamPathValid: true, UpdateChecked: true, OnlineVersion: "v1", PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1"}, text: "补丁已是最新"},
		{name: "a new patch", facts: app.Facts{SteamPathValid: true, UpdateChecked: true, OnlineVersion: "v1"}, text: "发现新补丁", download: true},
		{name: "a failed check", facts: app.Facts{SteamPathValid: true, UpdateFailed: true}, text: "检查失败"},
		{name: "a moved client", facts: app.Facts{SteamPathValid: true, UpdateChecked: true, OnlineVersion: "v2", SnapshotBuild: "1", ClientBuild: "2", PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1"}, text: "Steam 客户端已更新，补丁可能不适配", download: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tt := shown(t, test.facts)
			if test.text != "" && !tt.HasText(test.text) {
				t.Errorf("texts %q without %q", tt.Texts(), test.text)
			}
			if test.text == "" && (tt.HasText("补丁已是最新") || tt.HasText("发现新补丁") || tt.HasText("检查失败")) {
				t.Errorf("texts %q carry a conclusion before a check", tt.Texts())
			}
			if got := tt.HasText("下载并解压新版本"); got != test.download {
				t.Errorf("texts %q, download shown %v, want %v", tt.Texts(), got, test.download)
			}
		})
	}
}

func TestTheUpdateSlotRunsItsActions(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, UpdateChecked: true, OnlineVersion: "v1"}}
	var ran []string
	s.actions = view.Actions{
		CheckUpdate: func() { ran = append(ran, "check") },
		Download:    func() { ran = append(ran, "download") },
	}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("检查补丁更新"); err != nil {
		t.Fatalf("click the check button: %v", err)
	}
	if err := tt.Click("下载并解压新版本"); err != nil {
		t.Fatalf("click the download button: %v", err)
	}
	if !slices.Equal(ran, []string{"check", "download"}) {
		t.Errorf("ran %v, want the check then the download", ran)
	}
}

func TestABusyCommandLeavesTheUpdateButtonsInert(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, UpdateChecked: true, OnlineVersion: "v1", Busy: app.BusyApplyPatch}}
	ran := 0
	s.actions = view.Actions{
		CheckUpdate: func() { ran++ },
		Download:    func() { ran++ },
	}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("检查补丁更新"); err != nil {
		t.Fatalf("click the check button: %v", err)
	}
	if err := tt.Click("下载并解压新版本"); err != nil {
		t.Fatalf("click the download button: %v", err)
	}
	if ran != 0 {
		t.Errorf("ran %d actions while another command was busy", ran)
	}
}

func TestTheMainButtonRunsTheApply(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, PatchDownloaded: true, PatchComplete: true}}
	ran := 0
	s.actions = view.Actions{Apply: func() { ran++ }}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("应用补丁并启动 Steam"); err != nil {
		t.Fatalf("click the main button: %v", err)
	}
	if ran != 1 {
		t.Errorf("ran %d applies, want one", ran)
	}

	s.facts.Busy = app.BusyApplyPatch
	tt.Frame()
	if err := tt.Click("应用补丁并启动 Steam"); err != nil {
		t.Fatalf("click the main button while busy: %v", err)
	}
	if ran != 1 {
		t.Errorf("ran %d applies, the busy button is inert", ran)
	}

	s.facts.Busy = app.Idle
	s.facts.PatchComplete = false
	tt.Frame()
	if err := tt.Click("应用补丁并启动 Steam"); err != nil {
		t.Fatalf("click the main button with incomplete files: %v", err)
	}
	if ran != 1 {
		t.Errorf("ran %d applies, incomplete local files disable the button", ran)
	}
}

func TestAConflictingFileOffersTheRemoval(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, PatchDownloaded: true, Conflict: "version.dll"}}
	removed := 0
	s.actions = view.Actions{RemoveConflict: func() { removed++ }}
	tt := ui.NewTester(s.view, 480, 360)
	if !tt.HasText("检测到 version.dll，需先移除") {
		t.Fatalf("texts %q, want the conflicting file", tt.Texts())
	}
	if err := tt.Click("移除并部署"); err != nil {
		t.Fatalf("click the removal: %v", err)
	}
	if removed != 1 {
		t.Errorf("ran %d removals, want one", removed)
	}

	clear := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, PatchDownloaded: true}}
	if ui.NewTester(clear.view, 480, 360).HasText("移除并部署") {
		t.Error("the removal is offered without a conflict")
	}
}
