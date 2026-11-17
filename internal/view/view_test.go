package view_test

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/view"
)

type screen struct {
	facts app.Facts
}

func (s *screen) view(c *ui.Context) {
	view.Main(c, app.Derive(s.facts))
}

func shown(t *testing.T, facts app.Facts) *ui.Tester {
	t.Helper()
	s := &screen{facts: facts}
	return ui.NewTester(s.view, 480, 360)
}

func TestMainDerivesOneStateAndAtMostOneExplanation(t *testing.T) {
	invalid := []string{"Steam 路径无效", "请在设置中指定有效的 Steam 安装路径"}
	notDownloaded := []string{"补丁未下载", "请点击「检查补丁更新」下载新版本"}
	with := func(word string, rest ...string) []string {
		return append([]string{word, "Steam 未运行"}, rest...)
	}
	busyWith := func(stage string) []string {
		return []string{stage, stage, "Steam 未运行"}
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
			texts: []string{"已应用", "Steam 正在运行"},
		},
		{
			name:  "the last result follows the steam slot",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, SteamRunning: true, Result: app.SteamRestarted},
			texts: []string{"已应用", "Steam 正在运行", "Steam 已重启"},
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
			texts: with("未应用"),
		},
		{
			name:  "download busy keeps the file fact",
			facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Busy: app.BusyDownloadPatch},
			texts: with("已应用"),
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
			facts: app.Facts{Lang: app.English, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true},
			texts: []string{"Applied", "Steam is not running"},
		},
		{
			name:  "english result",
			facts: app.Facts{Lang: app.English, SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true, Result: app.StartSteamFailed},
			texts: []string{"Applied", "Steam is not running", "Failed to start Steam"},
		},
		{
			name:  "english invalid path",
			facts: app.Facts{Lang: app.English, SteamPathValid: false},
			texts: []string{"Not applied", "Steam is not running", "Steam path is invalid", "Set a valid Steam installation path in settings"},
		},
		{
			name:  "english patch failed",
			facts: app.Facts{Lang: app.English, SteamPathValid: true, PatchDownloaded: true, PatchFailed: true},
			texts: []string{"Patch failed to apply", "Steam is not running"},
		},
		{
			name:  "english patch not downloaded",
			facts: app.Facts{Lang: app.English, SteamPathValid: true},
			texts: []string{"Not applied", "Steam is not running", "Patch not downloaded", "Click Check for patch updates to download a new version"},
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
	s := &screen{facts: app.Facts{SteamPathValid: true, Deployment: app.Applied, PatchDownloaded: true}}
	tt := ui.NewTester(s.view, 480, 360)
	if !tt.HasText("已应用") {
		t.Fatalf("texts %q before the switch", tt.Texts())
	}
	s.facts.Lang = app.English
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
