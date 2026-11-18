package view_test

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/source"
	"opensteamtool-gui-mygo/internal/view"
)

func sourceDialog(t *testing.T, saved source.Source) (*screen, *ui.Tester) {
	t.Helper()
	s := &screen{facts: app.Facts{Locale: "zh-CN"}}
	s.state.Source = saved
	s.state.SourceChoice = view.ChoiceOf(saved)
	s.state.Entry = source.EntryDefault
	s.actions = view.Actions{
		SetLanguage: func(app.Language) {},
		CommitPath:  func(string) bool { return true },
		Browse:      func() {},
		ApplySource: func(draft source.Draft) (source.Source, bool) {
			resolved, err := draft.Resolve()
			if err != nil {
				return source.Source{}, false
			}
			return source.OrDefault(resolved), true
		},
		SetEntry: func(source.Entry) {},
	}
	tt := ui.NewTester(s.view, 520, 640)
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("click the gear: %v", err)
	}
	return s, tt
}

func openPatchSource(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click("补丁源"); err != nil {
		t.Fatalf("open the patch source tab: %v", err)
	}
}

func TestThePatchSourceTabListsTheBuiltinsAndTheirLayout(t *testing.T) {
	_, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	for _, want := range []string{
		"官方源（长期未更新）",
		"社区维护",
		"社区维护（SteamDaddy 布局）",
		"自定义仓库",
		"文件布局",
		"OpenSteamTool 布局",
	} {
		if !tt.HasText(want) {
			t.Errorf("the patch source tab shows %q without %q", tt.Texts(), want)
		}
	}
	if tt.HasText("应用并下载") {
		t.Errorf("the apply button shows for an untouched draft: %q", tt.Texts())
	}
}

func TestPickingAnotherSourceOffersTheApplyAndAppliesIt(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	if err := tt.Click("社区维护（SteamDaddy 布局）"); err != nil {
		t.Fatalf("pick the SteamDaddy source: %v", err)
	}
	tt.Frame()
	if !tt.HasText("应用并下载") {
		t.Fatalf("the draft differs without an apply button: %q", tt.Texts())
	}
	if !tt.HasText("SteamDaddy 布局") {
		t.Errorf("the layout line does not follow the draft: %q", tt.Texts())
	}
	if err := tt.Click("应用并下载"); err != nil {
		t.Fatalf("apply the draft: %v", err)
	}
	if s.state.Source.Key != source.KeySteamDaddy {
		t.Errorf("the applied source is %q, want the SteamDaddy one", s.state.Source.Key)
	}
	tt.Frame()
	if tt.HasText("应用并下载") {
		t.Errorf("the apply button stayed after applying: %q", tt.Texts())
	}
}

func TestARejectedCustomAddressShowsTheInlineError(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	if err := tt.Click("自定义仓库"); err != nil {
		t.Fatalf("pick the custom repository: %v", err)
	}
	if err := tt.Click("自定义源需 github.com 仓库地址（owner/repo）"); err != nil {
		t.Fatalf("focus the address field: %v", err)
	}
	tt.Type("github.com/someone/patch")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if !tt.HasText("地址无效：需要 github.com 仓库地址") {
		t.Errorf("a rejected address shows no inline error: %q", tt.Texts())
	}
	if s.state.Source.Key != source.KeyCommunity {
		t.Errorf("a rejected address changed the saved source to %q", s.state.Source.Key)
	}
	if tt.HasText("应用并下载") {
		t.Errorf("a rejected address offered an apply: %q", tt.Texts())
	}
}

func TestACustomAddressMatchingABuiltinCollapsesToIt(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	if err := tt.Click("自定义仓库"); err != nil {
		t.Fatalf("pick the custom repository: %v", err)
	}
	if err := tt.Click("自定义源需 github.com 仓库地址（owner/repo）"); err != nil {
		t.Fatalf("focus the address field: %v", err)
	}
	tt.Type("https://github.com/huanyuejue/OpenSteamTool/releases/latest")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if s.state.SourceChoice != source.KeyCommunity {
		t.Errorf("the address collapsed to %q, want the community source", s.state.SourceChoice)
	}
	if tt.HasText("应用并下载") {
		t.Errorf("the collapsed address still offers an apply: %q", tt.Texts())
	}
	if tt.HasText("地址无效：需要 github.com 仓库地址") {
		t.Errorf("a valid address reports an error: %q", tt.Texts())
	}
}

func TestACustomAddressCanBeApplied(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	if err := tt.Click("自定义仓库"); err != nil {
		t.Fatalf("pick the custom repository: %v", err)
	}
	if err := tt.Click("自定义源需 github.com 仓库地址（owner/repo）"); err != nil {
		t.Fatalf("focus the address field: %v", err)
	}
	tt.Type("https://github.com/someone/patch")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()

	if !tt.HasText("应用并下载") {
		t.Fatalf("a valid custom address offers no apply: %q", tt.Texts())
	}
	if err := tt.Click("应用并下载"); err != nil {
		t.Fatalf("apply the custom address: %v", err)
	}
	if s.state.Source.Key != "https://github.com/someone/patch" {
		t.Errorf("the applied source is %q, want the typed address", s.state.Source.Key)
	}
	if s.state.SourceCustom != "https://github.com/someone/patch" {
		t.Errorf("the address field reads %q after applying", s.state.SourceCustom)
	}
	tt.Frame()
	if tt.HasText("应用并下载") {
		t.Errorf("the apply button stayed after applying the address: %q", tt.Texts())
	}
}

func TestTheSteamTabOffersBothEntriesForTheOpenSteamToolLayout(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	var picked []source.Entry
	s.actions.SetEntry = func(entry source.Entry) { picked = append(picked, entry) }

	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	for _, want := range []string{"注入入口", "默认", "早期隔离"} {
		if !tt.HasText(want) {
			t.Errorf("the Steam tab shows %q without %q", tt.Texts(), want)
		}
	}
	if err := tt.Click("早期隔离"); err != nil {
		t.Fatalf("pick the early entry: %v", err)
	}
	if !slices.Equal(picked, []source.Entry{source.EntryEarly}) {
		t.Errorf("picked %v, want the early entry", picked)
	}
	if s.state.Entry != source.EntryEarly {
		t.Errorf("the entry reads %v, want the early one", s.state.Entry)
	}
}

func TestTheSteamTabHidesTheEntryForTheSteamDaddyLayout(t *testing.T) {
	steamDaddy := source.Source{Key: source.KeySteamDaddy, Repository: "Contrary-v2/SteamDaddy-Dll", Layout: source.LayoutSteamDaddy}
	_, tt := sourceDialog(t, steamDaddy)
	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	for _, hidden := range []string{"注入入口", "早期隔离"} {
		if tt.HasText(hidden) {
			t.Errorf("the SteamDaddy layout shows %q: %q", hidden, tt.Texts())
		}
	}
}

func TestTheOfficialSourceShowsTheEarlyEntryAsUnavailable(t *testing.T) {
	official := source.Source{Key: source.KeyOfficial, Repository: "OpenSteam001/OpenSteamTool", Layout: source.LayoutOpenSteamTool}
	s, tt := sourceDialog(t, official)
	picked := 0
	s.actions.SetEntry = func(source.Entry) { picked++ }

	if err := tt.Click("Steam 设置"); err != nil {
		t.Fatalf("open the Steam tab: %v", err)
	}
	info, ok := tt.Access("早期隔离")
	if !ok {
		t.Fatalf("the early entry is hidden rather than unavailable: %q", tt.Texts())
	}
	if !info.Disabled {
		t.Errorf("the early entry reads %+v, want it disabled", info)
	}
	_ = tt.Click("早期隔离")
	if picked != 0 {
		t.Errorf("an unavailable entry was picked %d times", picked)
	}
}

func TestTheSourceDraftSurvivesATabSwitch(t *testing.T) {
	s, tt := sourceDialog(t, source.Community())
	openPatchSource(t, tt)
	if err := tt.Click("社区维护（SteamDaddy 布局）"); err != nil {
		t.Fatalf("pick the SteamDaddy source: %v", err)
	}
	if err := tt.Click("通用"); err != nil {
		t.Fatalf("switch to the general tab: %v", err)
	}
	openPatchSource(t, tt)
	tt.Frame()

	if s.state.SourceChoice != source.KeySteamDaddy {
		t.Errorf("the draft reads %q, want the picked source", s.state.SourceChoice)
	}
	if !tt.HasText("应用并下载") {
		t.Errorf("the draft was lost on a tab switch: %q", tt.Texts())
	}
}
