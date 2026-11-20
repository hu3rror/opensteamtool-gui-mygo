package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
)

func wizardSession(t *testing.T, host rt.Runtime, cfg config.Config) *session {
	t.Helper()
	return newSession(host, noSleep, "zh-CN", cfg, false, t.TempDir())
}

func TestTheRerunStartsFromTheSavedValues(t *testing.T) {
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: steam, Language: app.LanguageEnglish, Theme: app.ThemeDark}, true, t.TempDir())

	s.actions.WizardRerun()
	texts := shown(t, s)
	if !slices.Contains(texts, "Changes every word in the window, the dialogs and the tray menu.") || !slices.Contains(texts, "Step 1 of 4") {
		t.Fatalf("texts = %q, want the wizard on the language step", texts)
	}
	if s.uiState.WizardLanguage != int(app.LanguageEnglish) || s.uiState.WizardTheme != int(app.ThemeDark) {
		t.Fatalf("drafts = %d/%d, want the saved English and dark theme", s.uiState.WizardLanguage, s.uiState.WizardTheme)
	}
	if s.uiState.PathDraft != steam {
		t.Errorf("path draft = %q, want the saved Steam path", s.uiState.PathDraft)
	}
}

func TestFinishingTheWizardReturnsToTheMainScreen(t *testing.T) {
	s := wizardSession(t, &rt.Fake{}, config.Config{})
	if !slices.Contains(shown(t, s), "首次运行向导") {
		t.Fatalf("texts = %q, want the wizard", shown(t, s))
	}
	s.actions.WizardSkip()
	if slices.Contains(shown(t, s), "首次运行向导") || !slices.Contains(shown(t, s), "未应用") {
		t.Fatalf("texts = %q, the skip must land on the main screen", shown(t, s))
	}
}

func runWizardDownload(t *testing.T, s *session) {
	t.Helper()
	if !s.startWizardDownload() {
		t.Fatal("the download was refused")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s.factsWithoutGate().Wizard.Download != app.WizardDownloading {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the download never finished")
}

func TestTheWizardDownloadFillsTheCacheFromItsOwnStep(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"xinput1_4.dll":     "loader",
		"dwmapi.dll":        "entry",
	})
	host := &rt.Fake{
		Responses: map[string][]byte{
			communityURL():                           releaseBody("v1.2.0", "https://example.test/patch-release.zip"),
			"https://example.test/patch-release.zip": archive,
		},
	}
	s := wizardSession(t, host, config.Config{})
	s.setWizardStep(app.WizardDownload)

	runWizardDownload(t, s)

	facts := s.factsWithoutGate()
	if facts.Wizard.Download != app.WizardDownloaded {
		t.Fatalf("the download step is %v, want a finished download", facts.Wizard.Download)
	}
	if !facts.PatchDownloaded || !facts.PatchComplete {
		t.Errorf("facts = %+v, want the cache filled", facts)
	}
	if _, err := os.Stat(filepath.Join(patch.Dir(s.dir, source.Community()), "OpenSteamTool.dll")); !errors.Is(err, nil) {
		t.Errorf("the cache is missing the payload: %v", err)
	}
	if !slices.Contains(shown(t, s), "补丁已下载，无需再下载。") {
		t.Errorf("texts = %q, want the downloaded step", shown(t, s))
	}
}

func TestTheWizardDownloadFailureStaysInTheStep(t *testing.T) {
	s := wizardSession(t, &rt.Fake{}, config.Config{})
	s.setWizardStep(app.WizardDownload)
	runWizardDownload(t, s)

	facts := s.factsWithoutGate()
	if facts.Wizard.Download != app.WizardDownloadFailed {
		t.Fatalf("the download step is %v, want a failure", facts.Wizard.Download)
	}
	if facts.Wizard.Failure != app.NetworkFailed {
		t.Errorf("failure = %v, want the network failure", facts.Wizard.Failure)
	}
	texts := shown(t, s)
	if !slices.Contains(texts, "补丁下载失败：网络请求失败") || !slices.Contains(texts, "重试") {
		t.Errorf("texts = %q, want the failure and its retry", texts)
	}
}

func TestAChoiceAppliesAtOnceAndIsSavedOnlyAtTheEnd(t *testing.T) {
	dir := t.TempDir()
	saves := 0
	s := wizardSession(t, &rt.Fake{}, config.Config{})
	s.persist = func(cfg config.Config) error {
		saves++
		return config.Save(dir, cfg)
	}

	s.actions.WizardLanguage(app.LanguageEnglish)
	if !slices.Contains(shown(t, s), "Changes every word in the window, the dialogs and the tray menu.") {
		t.Fatalf("texts = %q, the language did not apply at once", shown(t, s))
	}
	if s.factsWithoutGate().Wizard.Step != app.WizardLanguage {
		t.Fatal("picking a language advanced the wizard")
	}
	if saves != 0 {
		t.Fatalf("%d saves before the wizard ended, want none", saves)
	}

	s.actions.WizardSkip()
	if saves != 1 {
		t.Fatalf("%d saves at the end, want one", saves)
	}
	cfg, _, err := config.Load(dir, &rt.Fake{})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != app.LanguageEnglish {
		t.Errorf("saved language = %v, want the picked English", cfg.Language)
	}
}

func TestAnUnvalidatedPathIsNeverSaved(t *testing.T) {
	steam := t.TempDir()
	dir := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: steam, Language: app.LanguageChinese}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	s.tray = true
	s.actions.WizardRerun()
	s.actions.WizardNext()
	s.actions.WizardNext()
	s.uiState.PathDraft = filepath.Join(steam, "missing")
	if s.closeWindow() {
		t.Fatal("the close ended the process")
	}
	cfg, _, err := config.Load(dir, host)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SteamPath != steam {
		t.Errorf("saved path = %q, want the previously saved one", cfg.SteamPath)
	}
}

func TestTheDetectedSteamPathFillsTheEmptyDraft(t *testing.T) {
	base := t.TempDir()
	steam := filepath.Join(base, "Steam")
	host := &rt.Fake{Files: map[string]bool{filepath.Join(steam, "steam.exe"): true}}
	lookup := func(name string) string {
		if name == "ProgramFiles(x86)" {
			return base
		}
		return ""
	}
	noRegistry := func(string, string) string { return "" }
	if got := detectSteamPath(host, lookup, noRegistry); got != steam {
		t.Fatalf("detected %q, want %q", got, steam)
	}
	if got := detectSteamPath(&rt.Fake{}, lookup, noRegistry); got != "" {
		t.Errorf("detected %q without a Steam program", got)
	}
}

func TestTheRegistryAnswersTheSteamPathBeforeProgramFiles(t *testing.T) {
	base, registrySteam := t.TempDir(), t.TempDir()
	host := &rt.Fake{Files: map[string]bool{
		filepath.Join(registrySteam, "steam.exe"): true,
		filepath.Join(base, "Steam", "steam.exe"): true,
	}}
	registry := func(key, value string) string {
		if key == `Software\Valve\Steam` && value == "SteamPath" {
			return filepath.FromSlash(registrySteam)
		}
		return ""
	}
	want := filepath.FromSlash(registrySteam)
	if got := detectSteamPath(host, func(string) string { return base }, registry); got != want {
		t.Errorf("detected %q, want the registry path %q", got, want)
	}
}

func TestARegistryPathWithoutSteamIsSkipped(t *testing.T) {
	base, live, gone := t.TempDir(), t.TempDir(), t.TempDir()
	host := &rt.Fake{Files: map[string]bool{filepath.Join(live, "steam.exe"): true}}
	registry := func(key, value string) string {
		switch key {
		case `Software\Valve\Steam`:
			return gone
		case `HKLM\SOFTWARE\WOW6432Node\Valve\Steam`:
			return live
		}
		return ""
	}
	if got := detectSteamPath(host, func(string) string { return base }, registry); got != live {
		t.Errorf("detected %q, want %q", got, live)
	}
}

func TestDetectingTheSteamPathSavesWhatItFound(t *testing.T) {
	steam := t.TempDir()
	dir := t.TempDir()
	host := &rt.Fake{
		Dirs:  map[string]bool{steam: true},
		Files: map[string]bool{filepath.Join(steam, "steam.exe"): true},
	}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageChinese}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	s.registry = func(string, string) string { return steam }

	s.actions.Detect()

	if got := s.uiState.PathDraft; got != steam {
		t.Errorf("the field holds %q, want %q", got, steam)
	}
	if s.uiState.PathNotFound {
		t.Error("the not-found hint stays after a detection that found Steam")
	}
	cfg, _, err := config.Load(dir, host)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SteamPath != steam {
		t.Errorf("saved path = %q, want %q", cfg.SteamPath, steam)
	}
}

func TestAWizardThatFindsSteamEnablesTheNextStep(t *testing.T) {
	steam := t.TempDir()
	host := &rt.Fake{
		Dirs:  map[string]bool{steam: true},
		Files: map[string]bool{filepath.Join(steam, "steam.exe"): true},
	}
	steamRegistry = func(string, string) string { return steam }
	t.Cleanup(func() { steamRegistry = registryString })

	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageChinese}, false, t.TempDir())
	s.actions.WizardNext()
	s.actions.WizardNext()

	facts := s.snapshot()
	shown := app.Derive(facts)
	if shown.Wizard.Step != app.WizardPath {
		t.Fatalf("step = %v, want the path step", shown.Wizard.Step)
	}
	if shown.Wizard.NextDisabled {
		t.Error("the next step stays disabled after the wizard found Steam itself")
	}
	if got := s.config().SteamPath; got != steam {
		t.Errorf("path = %q, want the detected %q", got, steam)
	}
}

func TestDetectionWithoutSteamSaysSoAndChangesNothing(t *testing.T) {
	dir := t.TempDir()
	host := &rt.Fake{}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageChinese}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	s.registry = func(string, string) string { return "" }

	s.actions.Detect()

	if !s.uiState.PathNotFound {
		t.Error("detection found nothing without the not-found hint")
	}
	cfg, _, err := config.Load(dir, host)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.SteamPath != "" {
		t.Errorf("saved path = %q, want it untouched", cfg.SteamPath)
	}
}

func TestCommittingAPathClearsTheNotFoundHint(t *testing.T) {
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageChinese}, true, t.TempDir())
	s.uiState.PathNotFound = true

	if !s.commitSteamPath(steam) {
		t.Fatal("the commit failed")
	}
	if s.uiState.PathNotFound {
		t.Error("the not-found hint stays after a commit")
	}
}

func TestAFailedWizardDownloadCanBeRetried(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"xinput1_4.dll":     "loader",
		"dwmapi.dll":        "entry",
	})
	host := &rt.Fake{
		Responses: map[string][]byte{
			communityURL(): releaseBody("v1.2.0", "https://example.test/patch-release.zip"),
		},
	}
	s := wizardSession(t, host, config.Config{})
	s.setWizardStep(app.WizardDownload)
	runWizardDownload(t, s)
	if s.factsWithoutGate().Wizard.Download != app.WizardDownloadFailed {
		t.Fatalf("the first try did not fail")
	}

	host.Responses["https://example.test/patch-release.zip"] = archive
	runWizardDownload(t, s)
	facts := s.factsWithoutGate()
	if facts.Wizard.Download != app.WizardDownloaded || !facts.PatchComplete {
		t.Fatalf("the retry left %+v", facts.Wizard)
	}
	if facts.Result != app.NoResult {
		t.Errorf("the wizard wrote %v into the main screen result", facts.Result)
	}
}

func TestClosingTheWindowDuringTheWizardWithoutATrayEndsIt(t *testing.T) {
	s := wizardSession(t, &rt.Fake{}, config.Config{})
	if !s.closeWindow() {
		t.Fatal("the close was held back without a tray to hide to")
	}
}

func TestClosingTheWindowDuringTheWizardHidesItInstead(t *testing.T) {
	hidden := 0
	s := wizardSession(t, &rt.Fake{}, config.Config{})
	s.tray = true
	s.hide = func() { hidden++ }

	if s.closeWindow() {
		t.Fatal("the close ended the process")
	}
	if s.quitting {
		t.Error("the session is quitting")
	}
	if hidden != 1 {
		t.Errorf("hide ran %d times, want once", hidden)
	}
	if s.wizardOpen() {
		t.Error("the wizard is still open after the close")
	}
	if !slices.Contains(shown(t, s), "未应用") {
		t.Errorf("texts = %q, the close must land on the main screen", shown(t, s))
	}
}
