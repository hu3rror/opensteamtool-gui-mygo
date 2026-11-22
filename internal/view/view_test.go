package view_test

import (
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/mygo-breeze-ui/ui/breeze"

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
	removal := []string{"卸载补丁", "卸载补丁并重启 Steam"}
	exitRemoval := []string{"退出 Steam 并卸载补丁", "卸载补丁并重启 Steam"}
	removalEN := []string{"Remove the patch", "Remove the patch and restart Steam"}
	tests := []struct {
		name  string
		facts app.Facts
		texts []string
	}{
		{
			name:  "applied",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
			texts: with("已应用", removal...),
		},
		{
			name:  "not applied",
			facts: app.Facts{SteamPathValid: true, Deployment: app.NotApplied, PatchDownloaded: true},
			texts: with("未应用"),
		},
		{
			name:  "steam running",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, SteamRunning: true},
			texts: append([]string{"已应用", "Steam 正在运行", "应用补丁并启动 Steam", "检查补丁更新", "设置"}, exitRemoval...),
		},
		{
			name:  "the last result follows the steam slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, SteamRunning: true, Result: app.SteamRestarted},
			texts: append([]string{"已应用", "Steam 正在运行", "Steam 已重启", "应用补丁并启动 Steam", "检查补丁更新", "设置"}, exitRemoval...),
		},
		{
			name:  "a failed close lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.CloseSteamFailed},
			texts: append(with("已应用", "关闭 Steam 失败"), removal...),
		},
		{
			name:  "a missing steam.exe lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.MissingSteamProgram},
			texts: append(with("已应用", "未找到 steam.exe"), removal...),
		},
		{
			name:  "an applied patch lands in the result slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.PatchApplied},
			texts: append(with("已应用", "补丁已应用"), removal...),
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
			texts: append(busyWith("正在卸载补丁…"), removal...),
		},
		{
			name:  "start busy shows its stage",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyStartSteam},
			texts: append(busyWith("正在启动 Steam…"), removal...),
		},
		{
			name:  "stop busy shows its stage",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyStopSteam},
			texts: append(busyWith("正在关闭 Steam…"), removal...),
		},
		{
			name:  "update busy keeps the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.NotApplied, Busy: app.BusyCheckUpdate},
			texts: []string{"未应用", "Steam 未运行", "检查中…", "应用补丁并启动 Steam", "检查补丁更新", "设置"},
		},
		{
			name:  "tool check busy leaves the update slot alone",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyCheckTool},
			texts: append([]string{"已应用", "Steam 未运行", "应用补丁并启动 Steam", "检查补丁更新", "设置"}, removal...),
		},
		{
			name:  "download busy keeps the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyDownloadPatch},
			texts: append([]string{"已应用", "Steam 未运行", "应用补丁并启动 Steam", "检查补丁更新", "正在下载补丁…", "设置"}, removal...),
		},
		{
			name:  "a failed apply outranks the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, PatchFailed: true},
			texts: append(with("补丁应用失败"), removal...),
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
			texts: append(busyWith("正在应用补丁…"), removal...),
		},
		{
			name:  "english applied",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
			texts: append([]string{"Applied", "Steam is not running", "Settings", "Apply patch and start Steam", "Check for patch updates"}, removalEN...),
		},
		{
			name:  "english result",
			facts: app.Facts{Language: app.LanguageEnglish, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.StartSteamFailed},
			texts: append([]string{"Applied", "Steam is not running", "Failed to start Steam", "Settings", "Apply patch and start Steam", "Check for patch updates"}, removalEN...),
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
			texts: []string{"Not applied", "Steam is not running", "Settings", "Apply patch and start Steam", "Check for patch updates", "The patch is not downloaded.", "Click Check for patch updates to download a new version"},
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

func TestTheTrayMenuSpeaksBothLanguages(t *testing.T) {
	pairs := []struct {
		name    string
		copy    func(app.Lang) string
		chinese string
		english string
	}{
		{name: "restart Steam", copy: view.RestartSteam, chinese: "重启 Steam", english: "Restart Steam"},
		{name: "show", copy: view.Show, chinese: "显示", english: "Show"},
		{name: "exit", copy: view.Exit, chinese: "退出", english: "Exit"},
		{name: "open the Lua folder", copy: view.OpenLuaFolder, chinese: "打开 Lua 目录", english: "Open Lua folder"},
		{name: "the balloon", copy: view.TrayRunning, chinese: "仍在托盘运行，点击图标可显示窗口", english: "Still running in the tray. Click the icon to show the window."},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			if got := pair.copy(app.Chinese); got != pair.chinese {
				t.Errorf("the Chinese copy is %q, want %q", got, pair.chinese)
			}
			if got := pair.copy(app.English); got != pair.english {
				t.Errorf("the English copy is %q, want %q", got, pair.english)
			}
		})
	}
}

func TestTheExitConfirmationAsksBeforeEndingTheProcess(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied, SteamRunning: true, Pending: app.ConfirmExit}}
	var answered []string
	s.actions = view.Actions{
		Confirm: func() { answered = append(answered, "yes") },
		Cancel:  func() { answered = append(answered, "no") },
	}
	tt := ui.NewTester(s.view, 480, 360)
	for _, label := range []string{"确认", "确定退出？进行中的操作会中止。", "是", "否"} {
		if !tt.HasText(label) {
			t.Errorf("texts %q, want %q in the dialog", tt.Texts(), label)
		}
	}
	if err := tt.Click("是"); err != nil {
		t.Fatalf("click yes: %v", err)
	}
	if !slices.Equal(answered, []string{"yes"}) {
		t.Errorf("answered %v, want the yes", answered)
	}
}

func clickButton(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	target, ok := tt.AccessByRole(ui.RoleButton, label)
	if !ok || target.Bounds.W == 0 {
		t.Fatalf("no button named %q", label)
	}
	tt.ClickAt(target.Bounds.X+target.Bounds.W/2, target.Bounds.Y+target.Bounds.H/2)
}

func clickField(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	target, ok := tt.AccessByRole(ui.RoleTextField, label)
	if !ok || target.Bounds.W == 0 {
		t.Fatalf("no field named %q", label)
	}
	tt.ClickAt(target.Bounds.X+target.Bounds.W/2, target.Bounds.Y+target.Bounds.H/2)
}

func dialog(t *testing.T) (*screen, *ui.Tester) {
	t.Helper()
	s := &screen{facts: app.Facts{Locale: "zh-CN"}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
	}
	tt := ui.NewTester(s.view, 480, 700)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	return s, tt
}

func TestTheSettingsTabsBecomeOneDropdownWhenTheWindowIsNarrow(t *testing.T) {
	wide := &screen{facts: app.Facts{Locale: "zh-CN", WindowWidth: 420}}
	tt := ui.NewTester(wide.view, 420, 360)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("open settings: %v", err)
	}
	for _, label := range []string{"补丁源", "Lua 目录", "关于"} {
		if !tt.HasText(label) {
			t.Fatalf("wide texts %q, want the whole tab strip", tt.Texts())
		}
	}

	narrow := &screen{facts: app.Facts{Locale: "zh-CN", WindowWidth: 380}}
	tt = ui.NewTester(narrow.view, 380, 340)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("open settings: %v", err)
	}
	if tt.HasText("补丁源") {
		t.Fatalf("narrow texts %q, the whole strip still overflows", tt.Texts())
	}
	committed := 0
	narrow.actions = view.Actions{CommitPath: func(raw string) bool { committed++; return true }}
	narrow.state.PathFocused = true
	tt.Frame()
	if err := tt.Click("通用"); err != nil {
		t.Fatalf("open the tab dropdown: %v", err)
	}
	if err := tt.Click("补丁源"); err != nil {
		t.Fatalf("choose the patch source tab: %v", err)
	}
	if narrow.state.Tab != int(app.TabPatchSource) {
		t.Fatalf("tab = %d, want the patch source tab", narrow.state.Tab)
	}
	if committed != 1 || narrow.state.PathFocused {
		t.Errorf("commits = %d focused %v, want the path draft committed as on the strip", committed, narrow.state.PathFocused)
	}
}

func TestEverySettingsPageFitsTheSmallestWindow(t *testing.T) {
	const width, height = 460, 700
	for _, locale := range []string{"zh-CN", "en-US"} {
		for health := app.HealthChecking; health <= app.HealthNetworkUnavailable; health++ {
			facts := app.Facts{
				Locale:          locale,
				SteamPathValid:  true,
				SteamPath:       `C:\Program Files (x86)\Steam`,
				Deployment:      app.Applied,
				PatchDownloaded: true,
				Health:          health,
				OnlineVersion:   "1.2.3",
				UpdateChecked:   true,
				ToolVersion:     "0.2.1",
				LuaRelative:     "config/lua",
				LuaDir:          `C:\Program Files (x86)\Steam\config\lua`,
				WindowWidth:     width,
			}
			closeLabel := "关闭"
			if locale == "en-US" {
				closeLabel = "Close"
			}
			mainScreen := map[string]bool{}
			for _, text := range ui.NewTester((&screen{facts: facts}).view, width, height).Texts() {
				mainScreen[text] = true
			}

			for _, tab := range app.Tabs() {
				s := &screen{facts: facts}
				s.state.SettingsOpen = true
				s.state.Tab = int(tab)
				tt := ui.NewTester(s.view, width, height)
				closeRow, ok := tt.Find(closeLabel)
				if !ok {
					t.Fatalf("%s tab %v shows no close button", locale, tab)
				}
				for _, text := range tt.Texts() {
					row, ok := tt.Find(text)
					if !ok || mainScreen[text] || text == closeLabel {
						continue
					}
					if row.Y+row.H > closeRow.Y {
						t.Errorf("%s health %v tab %v: %q ends at %.0f below the close button at %.0f, the smallest window clips it", locale, health, tab, text, row.Y+row.H, closeRow.Y)
					}
					if row.X+row.W > width {
						t.Errorf("%s health %v tab %v: %q reaches %.0f past the %d wide window", locale, health, tab, text, row.X+row.W, width)
					}
				}
			}
		}
	}
}

func TestTheGearOpensTheSettingsDialogWithTheFiveTabs(t *testing.T) {
	s, tt := dialog(t)
	if !s.state.SettingsOpen {
		t.Error("the dialog did not open")
	}
	for _, label := range []string{"通用", "补丁源", "Steam 设置", "Lua 目录", "关于", "关闭"} {
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

func TestTheGeneralTabOffersTheThemeAndReportsTheChoice(t *testing.T) {
	s, tt := dialog(t)
	s.state.Language = int(app.LanguageChinese)
	tt.Frame()
	var picked []app.Theme
	s.actions.SetTheme = func(theme app.Theme) { picked = append(picked, theme) }

	for _, label := range []string{"主题", "跟随系统"} {
		if !tt.HasText(label) {
			t.Errorf("the dialog shows %q without %q", tt.Texts(), label)
		}
	}
	if err := tt.Click("跟随系统"); err != nil {
		t.Fatalf("open the theme dropdown: %v", err)
	}
	for _, label := range []string{"深色", "浅色"} {
		if !tt.HasText(label) {
			t.Errorf("the open dropdown shows %q without %q", tt.Texts(), label)
		}
	}
	if err := tt.Click("深色"); err != nil {
		t.Fatalf("choose the dark theme: %v", err)
	}
	if !slices.Equal(picked, []app.Theme{app.ThemeDark}) {
		t.Errorf("picked %v, want the dark theme", picked)
	}
	if err := tt.Click("深色"); err != nil {
		t.Fatalf("reopen the theme dropdown: %v", err)
	}
	if err := tt.Click("浅色"); err != nil {
		t.Fatalf("choose the light theme: %v", err)
	}
	if !slices.Equal(picked, []app.Theme{app.ThemeDark, app.ThemeLight}) {
		t.Errorf("picked %v, want the light theme second", picked)
	}
	if s.state.Theme != int(app.ThemeLight) {
		t.Errorf("the tab shows theme %d, want the light one", s.state.Theme)
	}
}

func TestTheLanguageChoiceIsReportedAsSoonAsItIsPicked(t *testing.T) {
	s, tt := dialog(t)
	s.state.Language = int(app.LanguageChinese)
	tt.Frame()
	var picked []app.Language
	s.actions.SetLanguage = func(language app.Language) { picked = append(picked, language) }

	if err := tt.Click("简体中文"); err != nil {
		t.Fatalf("open the language dropdown: %v", err)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatalf("choose English: %v", err)
	}
	if !slices.Equal(picked, []app.Language{app.LanguageEnglish}) {
		t.Errorf("picked %v, want the English preference", picked)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatalf("reopen the language dropdown: %v", err)
	}
	if err := tt.Click("简体中文"); err != nil {
		t.Fatalf("choose Chinese: %v", err)
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
	clickField(t, tt, "Steam 安装路径")
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
	clickField(t, tt, "Steam 安装路径")
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
	clickField(t, tt, "Steam 安装路径")
	tt.Type(`D:\Steam`)
	clickButton(t, tt, "浏览…")

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
	clickField(t, tt, "Steam 安装路径")
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
	clickField(t, tt, "Steam 安装路径")
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
	clickButton(t, tt, "浏览…")
	if asked != 1 {
		t.Errorf("browse was asked %d times, want once", asked)
	}
}

func TestTheDialogKeepsTheTabItWasLeftOn(t *testing.T) {
	s, tt := dialog(t)
	if err := tt.Click("Lua 目录"); err != nil {
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

func TestTheSecondaryButtonsFollowTheDeploymentAndSteam(t *testing.T) {
	applied := app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied}
	stopped := shown(t, applied)
	for _, label := range []string{"卸载补丁", "卸载补丁并重启 Steam"} {
		if !slices.Contains(stopped.Texts(), label) {
			t.Errorf("texts %q, want %q", stopped.Texts(), label)
		}
	}
	if slices.Contains(stopped.Texts(), "退出 Steam 并卸载补丁") {
		t.Errorf("texts %q offer to exit a Steam that is not running", stopped.Texts())
	}

	running := applied
	running.SteamRunning = true
	shownRunning := shown(t, running)
	for _, label := range []string{"退出 Steam 并卸载补丁", "卸载补丁并重启 Steam"} {
		if !slices.Contains(shownRunning.Texts(), label) {
			t.Errorf("texts %q, want %q", shownRunning.Texts(), label)
		}
	}
	if slices.Contains(shownRunning.Texts(), "卸载补丁") {
		t.Errorf("texts %q offer a removal that leaves the running Steam alone", shownRunning.Texts())
	}

	if notApplied := shown(t, app.Facts{Locale: "zh-CN", SteamPathValid: true}); slices.ContainsFunc(notApplied.Texts(), func(s string) bool { return strings.Contains(s, "卸载补丁") }) {
		t.Errorf("texts %q offer a removal without an applied patch", notApplied.Texts())
	}
}

func TestTheUninstallButtonsRunTheirActions(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied}}
	var ran []string
	s.actions = view.Actions{
		Uninstall:           func() { ran = append(ran, "remove") },
		UninstallAndRestart: func() { ran = append(ran, "restart") },
	}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("卸载补丁"); err != nil {
		t.Fatalf("click the removal: %v", err)
	}
	if err := tt.Click("卸载补丁并重启 Steam"); err != nil {
		t.Fatalf("click the removal and restart: %v", err)
	}

	s.facts.SteamRunning = true
	s.actions.ExitAndUninstall = func() { ran = append(ran, "exit") }
	tt.Frame()
	if err := tt.Click("退出 Steam 并卸载补丁"); err != nil {
		t.Fatalf("click the exit and removal: %v", err)
	}
	if !slices.Equal(ran, []string{"remove", "restart", "exit"}) {
		t.Errorf("ran %v, want the removal, then its restart, then the exit and removal", ran)
	}
}

func TestABusyCommandLeavesTheUninstallButtonsInert(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied, Busy: app.BusyApplyPatch}}
	ran := 0
	s.actions = view.Actions{
		Uninstall:           func() { ran++ },
		UninstallAndRestart: func() { ran++ },
	}
	tt := ui.NewTester(s.view, 480, 360)
	_ = tt.Click("卸载补丁")
	_ = tt.Click("卸载补丁并重启 Steam")
	if ran != 0 {
		t.Errorf("ran %d removals while another command was busy", ran)
	}
}

func TestTheConfirmDialogAsksABeforeTheUninstall(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied, SteamRunning: true, Pending: app.ConfirmExitRemove}}
	var answered []string
	s.actions = view.Actions{
		Confirm: func() { answered = append(answered, "yes") },
		Cancel:  func() { answered = append(answered, "no") },
	}
	tt := ui.NewTester(s.view, 480, 360)
	for _, label := range []string{"确认", "Steam 正在运行。是否自动关闭 Steam 并继续？", "是", "否"} {
		if !tt.HasText(label) {
			t.Errorf("texts %q, want %q in the dialog", tt.Texts(), label)
		}
	}
	if err := tt.Click("是"); err != nil {
		t.Fatalf("click yes: %v", err)
	}
	tt.Frame()
	if err := tt.Click("否"); err != nil {
		t.Fatalf("click no: %v", err)
	}
	if !slices.Equal(answered, []string{"yes", "no"}) {
		t.Errorf("answered %v, want the yes then the no", answered)
	}
}

func TestTheConfirmDialogStaysUntilItIsAnswered(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Deployment: app.Applied, SteamRunning: true, Pending: app.ConfirmRemoveAndRestart}}
	answered := 0
	s.actions = view.Actions{
		Confirm: func() { answered++ },
		Cancel:  func() { answered++ },
	}
	tt := ui.NewTester(s.view, 480, 360)
	tt.ClickAt(2, 2)
	tt.Frame()
	if !tt.HasText("确认") {
		t.Errorf("texts %q, the dialog closed on a click outside", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if !tt.HasText("确认") {
		t.Errorf("texts %q, the dialog closed on Escape", tt.Texts())
	}
	if answered != 0 {
		t.Errorf("answered %d times, want the question left open", answered)
	}
}

func TestTheRiskLineShowsOnlyForAnUnadaptedOrMissingCore(t *testing.T) {
	healths := []app.Health{
		app.HealthChecking,
		app.HealthCompatibleCached,
		app.HealthAdaptedNotCached,
		app.HealthOfflineReady,
		app.HealthNotAdapted,
		app.HealthMissing,
		app.HealthNetworkUnavailable,
	}
	for _, health := range healths {
		tt := shown(t, app.Facts{SteamPathValid: true, Health: health})
		want := health == app.HealthNotAdapted || health == app.HealthMissing
		if got := tt.HasText("当前 Steam 尚未适配") || tt.HasText("未找到 Steam 核心文件"); !want {
			if got {
				t.Errorf("health %v raises the risk line %q", health, tt.Texts())
			}
		} else if !got {
			t.Errorf("health %v shows no risk line in %q", health, tt.Texts())
		}
		if got := tt.HasText("当前 Steam 尚未适配   设置 →") || tt.HasText("未找到 Steam 核心文件   设置 →"); got != want {
			t.Errorf("health %v shows the settings affordance %v, want %v", health, got, want)
		}
	}
}

func TestTheRiskLineOpensTheSteamTab(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthNotAdapted}}
	tt := ui.NewTester(s.view, 480, 360)
	if err := tt.Click("当前 Steam 尚未适配   设置 →"); err != nil {
		t.Fatalf("click the risk line: %v", err)
	}
	if !s.state.SettingsOpen {
		t.Fatal("the risk line did not open the settings")
	}
	if s.state.Tab != int(app.TabSteam) {
		t.Errorf("the settings opened on tab %d, want the Steam tab", s.state.Tab)
	}
}

func steamTab(t *testing.T, facts app.Facts, a view.Actions) (*screen, *ui.Tester) {
	t.Helper()
	if facts.Locale == "" {
		facts.Locale = "zh-CN"
	}
	s := &screen{facts: facts}
	s.actions = a
	tt := ui.NewTester(s.view, 480, 720)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	return s, tt
}

func TestTheSteamTabOffersAutomaticDetection(t *testing.T) {
	detected := 0
	_, tt := steamTab(t, app.Facts{SteamPathValid: true}, view.Actions{
		CommitPath: func(string) bool { return true },
		Detect:     func() { detected++ },
	})
	if err := tt.Click("自动检测"); err != nil {
		t.Fatalf("click the detect button: %v", err)
	}
	if detected != 1 {
		t.Errorf("detection ran %d times, want once", detected)
	}
}

func TestTheSteamTabSaysWhenNothingWasFound(t *testing.T) {
	const hint = "未找到 Steam 安装目录，请手动选择"
	s, tt := steamTab(t, app.Facts{SteamPathValid: true}, view.Actions{
		CommitPath: func(string) bool { return true },
	})
	if tt.HasText(hint) {
		t.Fatalf("the hint shows before any detection: %q", tt.Texts())
	}
	s.state.PathNotFound = true
	tt.Frame()
	if !tt.HasText(hint) {
		t.Errorf("texts %q, want the not-found hint", tt.Texts())
	}
}

func compatDialog(t *testing.T) (*screen, *ui.Tester) {
	t.Helper()
	s := &screen{facts: app.Facts{Locale: "zh-CN"}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
	}
	tt := ui.NewTester(s.view, 480, 700)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	return s, tt
}

func TestTheCompatibilitySectionShowsTheHealthAndTheComponents(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{
		Locale:         "zh-CN",
		SteamPathValid: true,
		Health:         app.HealthAdaptedNotCached,
		Components: []app.Component{
			{DLL: "steamclient64.dll", Kind: "Pattern", Fingerprint: "0123456789ab", Health: app.HealthAdaptedNotCached},
			{DLL: "steamclient64.dll", Kind: "IPC", Health: app.HealthMissing},
		},
	}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	for _, want := range []string{"Steam 核心兼容性", "上游已适配 (未缓存)", "详细信息"} {
		if !tt.HasText(want) {
			t.Fatalf("texts %q without %q", tt.Texts(), want)
		}
	}
	if tt.HasText("steamclient64.dll（Pattern）") {
		t.Fatal("the components show before the details are expanded")
	}
	if err := tt.Click("详细信息"); err != nil {
		t.Fatalf("expand the details: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	for _, want := range []string{"steamclient64.dll（Pattern）", "0123456789ab", "steamclient64.dll（IPC）", "未找到核心 DLL"} {
		if !tt.HasText(want) {
			t.Fatalf("texts %q without %q", tt.Texts(), want)
		}
	}
}

func TestThePrewarmRunsFromTheBadgeAndFromTheDetails(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true, SteamCfg: true, Health: app.HealthAdaptedNotCached}
	ran := 0
	s.actions.Prewarm = func() { ran++ }
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	if err := tt.Click("预热离线缓存"); err != nil {
		t.Fatalf("click the prewarm button: %v", err)
	}
	if err := tt.Click("详细信息"); err != nil {
		t.Fatalf("expand the details: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	if err := tt.Click("一键缓存签名"); err != nil {
		t.Fatalf("click the cache action: %v", err)
	}
	if ran != 2 {
		t.Errorf("the prewarm ran %d times, want the badge row and the details action", ran)
	}
}

func TestTheCompatibilitySectionShowsThePrewarmState(t *testing.T) {
	tests := []struct {
		name  string
		facts app.Facts
		text  string
	}{
		{name: "running", facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthAdaptedNotCached, Prewarming: true}, text: "正在缓存..."},
		{name: "ready", facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthCompatibleCached, PrecacheReady: true}, text: "缓存已就绪"},
		{name: "failed", facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthAdaptedNotCached, PrecacheFailure: app.NetworkFailed}, text: "缓存预热失败：网络请求失败"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, tt := compatDialog(t)
			s.facts = test.facts
			tt.Frame()
			if err := tt.Click("Steam 设置"); err != nil {
				t.Fatalf("open the Steam tab: %v", err)
			}
			if !tt.HasText(test.text) {
				t.Errorf("texts %q without %q", tt.Texts(), test.text)
			}
		})
	}
}

func TestTheDetailsOfferTheCacheForAnUncachedComponent(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{
		Locale:         "zh-CN",
		SteamPathValid: true,
		Health:         app.HealthNetworkUnavailable,
		Components: []app.Component{
			{DLL: "steamclient64.dll", Kind: "Pattern", Fingerprint: "0123456789ab", Health: app.HealthAdaptedNotCached},
			{DLL: "steamui.dll", Kind: "Pattern", Health: app.HealthNetworkUnavailable},
		},
	}
	ran := 0
	s.actions.Prewarm = func() { ran++ }
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	if err := tt.Click("详细信息"); err != nil {
		t.Fatalf("expand the details: %v", err)
	}
	tt.Advance(500 * time.Millisecond)
	tt.Scroll(240, 560, 0, 200)
	tt.Frame()
	if err := tt.Click("一键缓存签名"); err != nil {
		t.Fatalf("click the cache action: %v", err)
	}
	if ran != 1 {
		t.Errorf("the prewarm ran %d times, want the details action", ran)
	}
}

func TestTheSteamTabStacksItsCardsSixteenApart(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthAdaptedNotCached}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	tt.Frame()
	edges := cardBorderRows(t, tt, 45, 415, 120, 700)
	if len(edges) != 6 {
		t.Fatalf("card edges at %v, want the 6 of three cards", edges)
	}
	for i := 2; i < len(edges); i += 2 {
		if reach := edges[i] - edges[i-1]; reach != 17 {
			t.Errorf("the cards at %v sit %d apart, want 16 DIP and the stroke below", edges, reach)
		}
	}
}

func TestTheHealthRowAndTheDetailsShareOneCard(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthAdaptedNotCached}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	tt.Frame()
	heading := bounds(t, tt, ui.RoleText, "Steam 核心兼容性")
	details, ok := tt.Find("详细信息")
	if !ok {
		t.Fatal("the details row is missing")
	}
	var edges []int
	for _, y := range cardBorderRows(t, tt, 45, 415, 120, 700) {
		if float32(y) > heading.Y-20 && float32(y) < details.Y+details.H+1 {
			edges = append(edges, y)
		}
	}
	if len(edges) != 2 {
		t.Fatalf("card edges at %v around the health row and the details, want the 2 of one card", edges)
	}
	if float32(edges[0]) >= heading.Y || float32(edges[1]) < details.Y {
		t.Errorf("the edges %v do not bracket the health row at %v and the details at %v", edges, heading.Y, details.Y)
	}
}

func TestTheSteamPathRowGivesTheFieldTheRoomAndEndsAtTheCardEdge(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	card := bounds(t, tt, ui.RoleGroup, "Steam 安装路径")
	field := bounds(t, tt, ui.RoleTextField, "Steam 安装路径")
	browse := bounds(t, tt, ui.RoleButton, "浏览…")
	detect := bounds(t, tt, ui.RoleButton, "自动检测")
	const cardPadding, rowGap = 16, 8
	if gap := field.X - (card.X + cardPadding); gap < 0 || gap > 1 {
		t.Errorf("the field starts %.1f past the card's text edge, want the card's edge", gap)
	}
	if gap := browse.X - (field.X + field.W); gap < rowGap || gap > rowGap+1 {
		t.Errorf("the field ends %.1f before the buttons, want the row's gap of %d", gap, rowGap)
	}
	if gap := card.X + card.W - cardPadding - (detect.X + detect.W); gap < 0 || gap > 1 {
		t.Errorf("the buttons end %.1f before the card's edge, want it", gap)
	}
}

func TestThePathRowLinesTheFieldUpWithItsButtons(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	field := bounds(t, tt, ui.RoleTextField, "Steam 安装路径")
	for _, name := range []string{"浏览…", "自动检测"} {
		button := bounds(t, tt, ui.RoleButton, name)
		if gap := button.Y - field.Y; gap < -1 || gap > 1 {
			t.Errorf("%q starts %.1f below the field, want the field's line", name, gap)
		}
		if gap := button.H - field.H; gap < -1 || gap > 1 {
			t.Errorf("%q is %.1f taller than the field, want the field's height", name, gap)
		}
	}
}

func TestEachInfoIconTellsItsNoteOnHover(t *testing.T) {
	s, tt := compatDialog(t)
	s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true, SteamCfg: true, Health: app.HealthAdaptedNotCached}
	tt.Frame()
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	notes := []string{
		"默认入口：布局 A 用 dwmapi.dll，布局 B 用 version.dll。\n早期隔离入口：布局 A 的 version.dll，默认不启用。",
		"steam.cfg 已锁定 Steam 客户端自动更新",
		"上游已适配，可缓存离线",
	}
	for _, note := range notes {
		if tipShown(tt, note) {
			t.Errorf("%q shows before the pointer rests on its icon", note)
		}
		icon := bounds(t, tt, ui.RoleImage, note)
		tt.Move(icon.X+icon.W/2, icon.Y+icon.H/2)
		tt.Advance(700 * time.Millisecond)
		tt.Frame()
		if !tipShown(tt, note) {
			t.Errorf("the icon named %q shows no tip once the pointer rests", note)
		}
		tt.Move(2, 2)
		tt.Advance(700 * time.Millisecond)
		tt.Frame()
	}
	reached := map[string]bool{}
	for range 40 {
		tt.Key(0, ui.KeyTab)
		tt.Frame()
		for _, note := range notes {
			if !tt.Focused(note) {
				continue
			}
			reached[note] = true
			tt.Advance(700 * time.Millisecond)
			tt.Frame()
			if !tipShown(tt, note) {
				t.Errorf("the focused icon named %q shows no tip", note)
			}
		}
	}
	for _, note := range notes {
		if !reached[note] {
			t.Errorf("no tab stop reaches the icon named %q", note)
		}
	}
}

func TestTheSectionHeadingsSitInsideTheirCards(t *testing.T) {
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true, Health: app.HealthCompatibleCached}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
		Detect:      func() {},
		Prewarm:     func() {},
	}
	tt := ui.NewTester(s.view, 460, 700)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	for _, name := range []string{"注入入口", "Steam 核心兼容性"} {
		heading := bounds(t, tt, ui.RoleText, name)
		if gap := heading.X - 57; gap < -1 || gap > 1 {
			t.Errorf("%q starts at %v, want the card's text edge at 57", name, heading.X)
		}
	}
}

func TestTheCompatibilityDetailsKeepTheirRowsInsideTheCard(t *testing.T) {
	s := &screen{facts: app.Facts{
		Locale:         "zh-CN",
		SteamPathValid: true,
		Health:         app.HealthCompatibleCached,
		Components: []app.Component{
			{DLL: "steamclient64.dll", Kind: "特征码", Fingerprint: "0123456789ab", Health: app.HealthCompatibleCached},
			{DLL: "steamclient64.dll", Kind: "IPC", Health: app.HealthCompatibleCached},
		},
	}}
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
		Detect:      func() {},
		Prewarm:     func() {},
	}
	tt := ui.NewTester(s.view, 460, 700)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	if err := tt.Click("详细信息"); err != nil {
		t.Fatalf("expand the details: %v", err)
	}
	tt.Advance(900 * time.Millisecond)
	tt.Frame()
	header, ok := tt.Find("详细信息")
	if !ok {
		t.Fatal("the details header is missing")
	}
	const panelEdge = 420
	for _, text := range tt.Texts() {
		if r, ok := tt.Find(text); ok && r.X >= 40 && r.Y >= header.Y+header.H && r.X+r.W > panelEdge {
			t.Errorf("%q at %v runs past the dialog's edge %v", text, r, panelEdge)
		}
	}
	label := bounds(t, tt, ui.RoleText, "steamclient64.dll（特征码）")
	fingerprint := bounds(t, tt, ui.RoleText, "0123456789ab")
	if fingerprint.Y <= label.Y+label.H {
		t.Errorf("the fingerprint sits on the label's line at y=%v, want its own line below the label's %v", fingerprint.Y, label.Y+label.H)
	}
}

func TestTheSteamCfgNoteAppearsOnlyWithTheFile(t *testing.T) {
	note := "steam.cfg 已锁定 Steam 客户端自动更新"
	for _, present := range []bool{false, true} {
		s, tt := compatDialog(t)
		s.facts = app.Facts{Locale: "zh-CN", SteamPathValid: true, SteamCfg: present}
		tt.Frame()
		if err := tt.Click("Steam 设置"); err != nil {
			t.Fatalf("open the Steam tab: %v", err)
		}
		if _, ok := tt.AccessByRole(ui.RoleImage, note); ok != present {
			t.Errorf("the steam.cfg icon shows %v, want %v", ok, present)
		}
	}
}

func TestTheMainScreenFitsTheSmallestWindowInEnglish(t *testing.T) {
	facts := app.Facts{
		Locale:          "en-US",
		SteamPathValid:  true,
		PatchDownloaded: true,
		SteamRunning:    true,
		Busy:            app.BusyApplyPatch,
		Health:          app.HealthOfflineReady,
		UpdateChecked:   true,
		OnlineVersion:   "1.2.3",
	}
	const width, height = 460, 700
	tt := ui.NewTester((&screen{facts: facts}).view, width, height)
	tt.Frame()
	for _, text := range tt.Texts() {
		r, ok := tt.Find(text)
		if !ok {
			continue
		}
		if r.X+r.W > width-2 || r.X < -0.5 || r.Y+r.H > height+0.5 || r.Y < -0.5 {
			t.Errorf("%q at %v leaves the %dx%d window", text, r, width, height)
		}
	}
	line := func(text string) float32 {
		t.Helper()
		if r, ok := tt.Find(text); ok {
			return r.H
		}
		t.Fatalf("no %q", text)
		return 0
	}
	if h := line("New patch found"); h > 20 {
		t.Errorf("the conclusion wraps to %v, want one line: the row is too narrow for its items", h)
	}
}

func tipShown(tt *ui.Tester, note string) bool {
	_, ok := tt.AccessByRole(ui.RoleTooltip, note)
	return ok
}

// cardBorderRows returns the y of every row between top and bottom that a card's stroke covers across x0 to x1, in the light theme compatDialog draws.
func cardBorderRows(t *testing.T, tt *ui.Tester, x0, x1, top, bottom int) []int {
	t.Helper()
	stroke := platformLook(t, false).Background.Mix(ui.Hex("#000000"), 0.059)
	img := image.Image(tt.Image())
	var rows []int
	for y := top; y < bottom; y++ {
		covered := 0
		for x := x0; x < x1; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if uint8(r>>8) == stroke.R && uint8(g>>8) == stroke.G && uint8(b>>8) == stroke.B {
				covered++
			}
		}
		if covered*10 > (x1-x0)*9 {
			rows = append(rows, y)
		}
	}
	return rows
}

func bounds(t *testing.T, tt *ui.Tester, role ui.Role, name string) ui.Rect {
	t.Helper()
	target, ok := tt.AccessByRole(role, name)
	if !ok || target.Bounds.W == 0 {
		t.Fatalf("no %v named %q", role, name)
	}
	return target.Bounds
}

func TestAToolCheckNeverDisablesThePatchButtons(t *testing.T) {
	tt := shown(t, app.Facts{SteamPathValid: true, Busy: app.BusyCheckTool})

	check, found := tt.Access("检查补丁更新")
	if !found {
		t.Fatal("the patch check is gone")
	}
	if check.Disabled {
		t.Error("the patch check waits for the tool check")
	}
}

func platformLook(t *testing.T, dark bool) *ui.Theme {
	t.Helper()
	var look *ui.Theme
	ui.NewTester(func(c *ui.Context) {
		look = breeze.ThemeOf(c, breeze.Current(), dark, ui.Color{})
	}, 100, 100)
	return look
}

type themeLook struct {
	dark                                     bool
	background, surface, text, accent, focus ui.Color
	fontSize, radius                         float32
}

func lookOf(t ui.Theme) themeLook {
	return themeLook{t.Dark, t.Background, t.Surface, t.Text, t.Accent, t.Focus, t.FontSize, t.Radius}
}

func platformTheme(t *testing.T, dark bool, prefs ui.Preferences) (want, got themeLook) {
	t.Helper()
	s := &screen{facts: app.Facts{Locale: "zh-CN", SteamPathValid: true}}
	tt := ui.NewTesterWith(func(c *ui.Context) {
		s.view(c)
		got = lookOf(*c.Theme())
		want = lookOf(*breeze.ThemeOf(c, breeze.Current(), dark, ui.Color{}))
	}, 480, 360, ui.TesterOptions{Dark: dark, Preferences: prefs})
	tt.Frame()
	return want, got
}

func TestTheViewDrawsInThePlatformTheme(t *testing.T) {
	lightWant, light := platformTheme(t, false, ui.Preferences{})
	darkWant, dark := platformTheme(t, true, ui.Preferences{})
	if light != lightWant || dark != darkWant {
		t.Errorf("the window draws in %+v and %+v, want the platform themes %+v and %+v", light, dark, lightWant, darkWant)
	}
	if light.background == dark.background || light.text == dark.text {
		t.Errorf("both appearances draw the same, the appearance setting is lost")
	}
	usual, _ := platformTheme(t, false, ui.Preferences{})
	chosen := ui.Preferences{Accent: ui.Hex("#8e44ad"), HighContrast: true, TextScale: 1.5}
	want, got := platformTheme(t, false, chosen)
	if got != want {
		t.Errorf("the window draws in %+v, want the platform theme %+v under the desktop's choices", got, want)
	}
	if got.accent == usual.accent {
		t.Errorf("accent %v, the window ignores the accent the desktop chose", got.accent)
	}
	if got.fontSize <= usual.fontSize {
		t.Errorf("font size %v, want more than the usual %v under a larger text size", got.fontSize, usual.fontSize)
	}
	if got.focus.A != 255 {
		t.Errorf("focus ring %v, want it opaque under the higher contrast setting", got.focus)
	}
}

func TestTheWindowDrawsItsOwnTitleBarWhenTheSystemOneIsHidden(t *testing.T) {
	for _, wizard := range []bool{false, true} {
		s := &screen{facts: app.Facts{Locale: "zh-CN", ToolVersion: "0.2.2"}}
		s.facts.Wizard.Open = wizard
		tt := ui.NewTester(s.view, 480, 360)
		if tt.HasText(view.Title()) {
			t.Fatalf("wizard %v: texts %q, a window with a system title bar draws none of its own", wizard, tt.Texts())
		}
		tt.SetTitleBar(ui.TitleBar{Height: 48, Right: 138})
		for _, want := range []string{view.Title(), "0.2.2"} {
			if !tt.HasText(want) {
				t.Errorf("wizard %v: texts %q, want %q in the title bar", wizard, tt.Texts(), want)
			}
		}
		if bar, ok := tt.AccessByRole(ui.RoleGroup, view.Title()); !ok || bar.Bounds.H == 0 {
			t.Fatalf("wizard %v: no title bar group named %q in %+v", wizard, view.Title(), tt.AccessAll())
		}
		tt.ClickAt(416, 20)
		if s.state.SettingsOpen || tt.HasText("补丁源") {
			t.Errorf("wizard %v: a press on the title bar reached the page", wizard)
		}
	}
}
