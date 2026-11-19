package app_test

import (
	"path/filepath"
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
)

func TestLangForFollowsTheLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   app.Lang
	}{
		{"zh-Hans", app.Chinese},
		{"zh-CN", app.Chinese},
		{"zh-Hant-TW", app.Chinese},
		{"ZH", app.Chinese},
		{"en-US", app.English},
		{"de", app.English},
		{"", app.English},
	}
	for _, test := range tests {
		if got := app.LangFor(test.locale); got != test.want {
			t.Errorf("LangFor(%q) = %v, want %v", test.locale, got, test.want)
		}
	}
}

func openSteamToolLayout() app.Layout {
	manifest := source.LayoutOpenSteamTool.Manifest()
	entry, _ := source.LayoutOpenSteamTool.EntryFile(source.EntryDefault)
	return app.Layout{Payload: manifest.Payload.Name, SecondaryLoader: manifest.Loader.Name, DefaultEntry: entry}
}

func TestDeploymentFollowsTheLayoutFiles(t *testing.T) {
	root := `C:\Steam`
	holding := func(names ...string) rt.Fake {
		files := map[string]bool{}
		for _, name := range names {
			files[filepath.Join(root, name)] = true
		}
		return rt.Fake{Files: files}
	}
	tests := []struct {
		name  string
		entry string
		files rt.Fake
		want  app.Deployment
	}{
		{
			name:  "default entry",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"),
			want:  app.Applied,
		},
		{
			name:  "early entry selected",
			entry: "version.dll",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "version.dll"),
			want:  app.Applied,
		},
		{
			name:  "missing payload",
			files: holding("xinput1_4.dll", "dwmapi.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "missing secondary loader",
			files: holding("OpenSteamTool.dll", "dwmapi.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "missing entry",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "only the entry that is not selected",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "version.dll"),
			want:  app.NotApplied,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := app.DeploymentOf(openSteamToolLayout(), test.entry, root, test.files); got != test.want {
				t.Errorf("DeploymentOf(%+v) = %v, want %v", test.files.Files, got, test.want)
			}
		})
	}
}

func TestDeploymentNeedsASteamRoot(t *testing.T) {
	if got := app.DeploymentOf(openSteamToolLayout(), "", "", everything{}); got != app.NotApplied {
		t.Errorf("DeploymentOf against no Steam root = %v, want %v", got, app.NotApplied)
	}
}

type everything struct{}

func (everything) IsFile(string) bool { return true }

func TestTheLanguagePreferenceResolvesAgainstTheLocale(t *testing.T) {
	tests := []struct {
		preference app.Language
		locale     string
		want       app.Lang
	}{
		{app.LanguageAuto, "zh-Hans", app.Chinese},
		{app.LanguageAuto, "en-GB", app.English},
		{app.LanguageAuto, "", app.English},
		{app.LanguageChinese, "en-US", app.Chinese},
		{app.LanguageEnglish, "zh-CN", app.English},
	}
	for _, test := range tests {
		if got := test.preference.Resolve(test.locale); got != test.want {
			t.Errorf("%v resolved against %q = %v, want %v", test.preference, test.locale, got, test.want)
		}
	}
}

func TestTheDerivedScreenCarriesTheResolvedLanguage(t *testing.T) {
	view := app.Derive(app.Facts{Language: app.LanguageChinese, Locale: "en-US"})
	if view.Lang != app.Chinese {
		t.Errorf("the screen reads %v, want Chinese", view.Lang)
	}
	view = app.Derive(app.Facts{Language: app.LanguageAuto, Locale: "zh-CN"})
	if view.Lang != app.Chinese {
		t.Errorf("an auto preference on a Chinese locale reads %v", view.Lang)
	}
}

func TestTheSettingsTabsAreInAFixedOrder(t *testing.T) {
	want := []app.Tab{app.TabGeneral, app.TabPatchSource, app.TabSteam, app.TabLua, app.TabAbout}
	if got := app.Tabs(); !slices.Equal(got, want) {
		t.Errorf("tabs = %v, want %v", got, want)
	}
}

func TestAnUnreadableConfigAsksForSetup(t *testing.T) {
	if view := app.Derive(app.Facts{NeedsSetup: true}); !view.NeedsSetup {
		t.Error("an unreadable config does not ask for setup")
	}
	if view := app.Derive(app.Facts{Language: app.LanguageAuto}); view.NeedsSetup {
		t.Error("a readable config asks for setup")
	}
}

func TestTheConflictReachesTheScreen(t *testing.T) {
	view := app.Derive(app.Facts{Conflict: "version.dll"})
	if view.Conflict != "version.dll" {
		t.Errorf("the screen carries %q, want the conflicting file", view.Conflict)
	}
	if blank := app.Derive(app.Facts{}); blank.Conflict != "" {
		t.Errorf("a screen without a conflict carries %q", blank.Conflict)
	}
}

func TestTheSecondaryRowFollowsTheDeploymentAndSteam(t *testing.T) {
	applied := app.Facts{SteamPathValid: true, Deployment: app.Applied}
	running := applied
	running.SteamRunning = true
	tests := []struct {
		name  string
		facts app.Facts
		want  []app.SecondaryAction
	}{
		{name: "no path", facts: app.Facts{Deployment: app.Applied}},
		{name: "not applied", facts: app.Facts{SteamPathValid: true}},
		{name: "applied with Steam stopped", facts: applied, want: []app.SecondaryAction{app.SecondaryUninstall, app.SecondaryUninstallAndRestart}},
		{name: "applied with Steam running", facts: running, want: []app.SecondaryAction{app.SecondaryExitAndUninstall, app.SecondaryUninstallAndRestart}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := app.Derive(test.facts).Secondary; !slices.Equal(got, test.want) {
				t.Errorf("the secondary row is %v, want %v", got, test.want)
			}
		})
	}
}

func TestAPendingConfirmationReachesTheScreen(t *testing.T) {
	if view := app.Derive(app.Facts{Pending: app.ConfirmExitRemove}); view.Pending != app.ConfirmExitRemove {
		t.Errorf("the screen carries %v, want the pending question", view.Pending)
	}
	if blank := app.Derive(app.Facts{}); blank.Pending != app.NoConfirmation {
		t.Errorf("a screen without a question carries %v", blank.Pending)
	}
}

func TestThePrimaryActionNeedsAPathAndCompleteFiles(t *testing.T) {
	ready := app.Facts{SteamPathValid: true, PatchDownloaded: true, PatchComplete: true}
	tests := []struct {
		name     string
		facts    app.Facts
		disabled bool
	}{
		{name: "ready", facts: ready},
		{name: "no path", facts: app.Facts{PatchDownloaded: true, PatchComplete: true}, disabled: true},
		{name: "no download", facts: app.Facts{SteamPathValid: true, PatchComplete: true}, disabled: true},
		{name: "incomplete files", facts: app.Facts{SteamPathValid: true, PatchDownloaded: true}, disabled: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := app.Derive(test.facts).PrimaryDisabled; got != test.disabled {
				t.Errorf("the primary action is disabled = %v, want %v", got, test.disabled)
			}
		})
	}
}
