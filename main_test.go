package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
)

func shown(t *testing.T, s *session) []string {
	t.Helper()
	return ui.NewTester(s.view, 480, 360).Texts()
}

func noSleep(time.Duration) {}

func newTestSession(t *testing.T, host rt.Runtime, root string) *session {
	t.Helper()
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto}, true, t.TempDir())
	return s
}

func TestTheSessionCarriesTheWindowWidthIntoTheView(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{}, false, t.TempDir())
	s.windowWidth = func() float32 { return 380 }
	if got := s.snapshot().WindowWidth; got != 380 {
		t.Fatalf("window width = %v, want 380", got)
	}
}

func TestWindowShowsTheDerivedScreenAndFollowsTheLanguage(t *testing.T) {
	s := &session{facts: app.Facts{Language: app.LanguageChinese, Locale: "zh-CN"}}

	chinese := ui.NewTester(s.view, 480, 360)
	if !slices.Contains(chinese.Texts(), "未应用") {
		t.Fatalf("texts = %q, want 未应用", chinese.Texts())
	}
	if !slices.Contains(chinese.Texts(), "Steam 路径无效") {
		t.Fatalf("texts = %q, want the invalid path explanation", chinese.Texts())
	}

	s.facts.Language = app.LanguageEnglish
	english := ui.NewTester(s.view, 480, 360)
	if !slices.Contains(english.Texts(), "Not applied") {
		t.Fatalf("texts = %q, want Not applied", english.Texts())
	}
}

func TestARealSteamDirectoryWithTheLayoutReadsAsApplied(t *testing.T) {
	steam := t.TempDir()
	for _, name := range []string{"steam.exe", "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		writeFile(t, filepath.Join(steam, name))
	}

	s := &session{facts: factsFor(app.LanguageChinese, "zh-CN", config.Config{SteamPath: steam}, rt.OS{}, t.TempDir())}
	if !slices.Contains(shown(t, s), "已应用") {
		t.Fatalf("texts = %q, want 已应用", shown(t, s))
	}
}

func TestARealSteamDirectoryWithoutTheLayoutReadsAsNotApplied(t *testing.T) {
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "steam.exe"))

	s := &session{facts: factsFor(app.LanguageChinese, "zh-CN", config.Config{SteamPath: steam}, rt.OS{}, t.TempDir())}
	texts := shown(t, s)
	if !slices.Contains(texts, "未应用") {
		t.Fatalf("texts = %q, want 未应用", texts)
	}
	if slices.Contains(texts, "Steam 路径无效") {
		t.Fatalf("texts = %q, the path is valid", texts)
	}
}

func steamHost(t *testing.T, running bool) (*rt.Fake, string) {
	t.Helper()
	root := t.TempDir()
	exe := filepath.Join(root, "steam.exe")
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{exe: true}}
	if running {
		host.Procs = []rt.Process{{PID: 100, Exe: exe}}
	}
	return host, root
}

func TestTheTrayRestartClosesThenStarts(t *testing.T) {
	host, root := steamHost(t, true)
	host.OnSpawn = func(call rt.Launch) {
		if slices.Equal(call.Args, []string{"-shutdown"}) {
			host.Procs = nil
			return
		}
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}
	s := newTestSession(t, host, root)
	frames := 0
	s.frame = func() { frames++ }

	if !s.restartSteam() {
		t.Fatal("the restart was refused while the gate was idle")
	}
	if len(host.Spawned) != 2 {
		t.Fatalf("spawned %d processes, want the shutdown signal and the new instance", len(host.Spawned))
	}
	if !slices.Equal(host.Spawned[0].Args, []string{"-shutdown"}) || host.Spawned[0].Dir != root {
		t.Fatalf("first spawn = %+v, want -shutdown in the Steam directory", host.Spawned[0])
	}
	if len(host.Spawned[1].Args) != 0 || host.Spawned[1].Dir != root {
		t.Fatalf("second spawn = %+v, want a bare start in the Steam directory", host.Spawned[1])
	}
	if len(host.Killed) != 0 {
		t.Fatalf("killed %v, the graceful signal was enough", host.Killed)
	}

	facts := s.snapshot()
	if facts.Result != app.SteamRestarted || !facts.SteamRunning || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want a restarted and running Steam with an idle gate", facts)
	}
	texts := shown(t, s)
	if !slices.Contains(texts, "Steam 已重启") || !slices.Contains(texts, "Steam 正在运行") {
		t.Fatalf("texts = %q, want the result and the running slot", texts)
	}
	if frames == 0 {
		t.Fatal("no frame was asked for")
	}
}

func TestARequestWhileBusyIsDroppedWithoutAQueue(t *testing.T) {
	host, root := steamHost(t, false)
	s := newTestSession(t, host, root)
	s.mu.Lock()
	if !s.gate.Try(app.BusyApplyPatch) {
		t.Fatal("the gate was busy")
	}
	s.mu.Unlock()

	if s.restartSteam() {
		t.Fatal("a request was taken while another kind was busy")
	}
	if len(host.Spawned) != 0 {
		t.Fatalf("spawned %d processes, want none", len(host.Spawned))
	}
	if facts := s.snapshot(); facts.Result != app.NoResult || facts.Busy != app.BusyApplyPatch {
		t.Fatalf("facts = %+v, want the busy kind unchanged", facts)
	}

	s.mu.Lock()
	s.gate.Release()
	s.mu.Unlock()
	host.OnSpawn = func(call rt.Launch) {
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}
	if !s.restartSteam() {
		t.Fatal("the gate stayed busy after it was released")
	}
	if len(host.Spawned) != 1 {
		t.Fatalf("spawned %d processes, want only the request made after the release", len(host.Spawned))
	}
}

func TestThePollFollowsASteamProcess(t *testing.T) {
	host, root := steamHost(t, false)
	s := newTestSession(t, host, root)
	if !slices.Contains(shown(t, s), "Steam 未运行") {
		t.Fatalf("texts = %q, want the stopped slot", shown(t, s))
	}
	if s.refresh() {
		t.Fatal("the poll reported a change without one")
	}

	host.Procs = []rt.Process{{PID: 100, Exe: filepath.Join(root, "steam.exe")}}
	if !s.refresh() {
		t.Fatal("the poll missed a started Steam")
	}
	if !slices.Contains(shown(t, s), "Steam 正在运行") {
		t.Fatalf("texts = %q, want the running slot", shown(t, s))
	}

	host.Procs = nil
	if !s.refresh() {
		t.Fatal("the poll missed a stopped Steam")
	}
	if !slices.Contains(shown(t, s), "Steam 未运行") {
		t.Fatalf("texts = %q, want the stopped slot", shown(t, s))
	}
}

func TestTheCloseFailureIsRecordedAndTheStageClears(t *testing.T) {
	host, root := steamHost(t, true)
	host.OnTerminate = func(int) {}
	s := newTestSession(t, host, root)
	if !s.restartSteam() {
		t.Fatal("the restart was refused while the gate was idle")
	}

	if len(host.Spawned) != 1 {
		t.Fatalf("spawned %d processes, want only the shutdown signal", len(host.Spawned))
	}
	facts := s.snapshot()
	if facts.Result != app.CloseSteamFailed || facts.Busy != app.Idle || !facts.SteamRunning {
		t.Fatalf("facts = %+v, want the close failure with an idle gate", facts)
	}
	texts := shown(t, s)
	if !slices.Contains(texts, "关闭 Steam 失败") {
		t.Fatalf("texts = %q, want the close failure", texts)
	}
	if slices.Contains(texts, "正在关闭 Steam…") {
		t.Fatalf("texts = %q, the stage must clear on failure", texts)
	}
}

func TestAStartWithoutTheSteamProgramIsRecorded(t *testing.T) {
	root := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	s := newTestSession(t, host, root)

	if !s.restartSteam() {
		t.Fatal("the start was refused while the gate was idle")
	}
	if len(host.Spawned) != 0 {
		t.Fatalf("spawned %d processes, there is no steam.exe", len(host.Spawned))
	}
	if facts := s.snapshot(); facts.Result != app.MissingSteamProgram || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the missing program with an idle gate", facts)
	}
	if !slices.Contains(shown(t, s), "未找到 steam.exe") {
		t.Fatalf("texts = %q, want the missing program", shown(t, s))
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestACommittedSteamPathSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "steam.exe"))
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}

	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	if !s.commitSteamPath(steam) {
		t.Fatal("a readable directory was refused")
	}

	cfg, readable, err := config.Load(dir, host)
	if err != nil {
		t.Fatal(err)
	}
	if !readable || cfg.SteamPath != steam {
		t.Fatalf("config = %+v readable = %v, want the committed path", cfg, readable)
	}

	restarted := newSession(host, noSleep, "zh-CN", cfg, readable, t.TempDir())
	texts := shown(t, restarted)
	if restarted.uiState.PathDraft != steam {
		t.Fatalf("draft = %q, want the committed path", restarted.uiState.PathDraft)
	}
	if slices.Contains(texts, "Steam 路径无效") {
		t.Fatalf("texts = %q, the committed path is valid", texts)
	}
}

func TestACommittedPathRefreshesTheDeploymentState(t *testing.T) {
	steam := t.TempDir()
	for _, name := range []string{"steam.exe", "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		writeFile(t, filepath.Join(steam, name))
	}
	s := newSession(rt.OS{}, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	s.persist = func(config.Config) error { return nil }

	if !slices.Contains(shown(t, s), "Steam 路径无效") {
		t.Fatalf("texts = %q, want an unset path first", shown(t, s))
	}
	if !s.commitSteamPath(steam) {
		t.Fatal("a readable directory was refused")
	}
	if facts := s.snapshot(); facts.Deployment != app.Applied {
		t.Fatalf("deployment = %v, want the committed path applied", facts.Deployment)
	}
	if !slices.Contains(shown(t, s), "已应用") {
		t.Fatalf("texts = %q, want the committed path applied on the main screen", shown(t, s))
	}
}

func TestAPathThatCannotBeReadIsRefused(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there")
	unreadable := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{unreadable: true}, Unreadable: map[string]bool{unreadable: true}}

	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	writes := 0
	s.persist = func(config.Config) error {
		writes++
		return nil
	}

	if s.commitSteamPath(missing) {
		t.Fatal("a directory that does not exist was taken")
	}
	if s.commitSteamPath(unreadable) {
		t.Fatal("a directory that cannot be read was taken")
	}
	if facts := s.snapshot(); facts.SteamPath != "" || facts.SteamPathValid {
		t.Fatalf("facts = %+v, want the path unchanged", facts)
	}
	if writes != 0 {
		t.Fatalf("the config was written %d times, want none", writes)
	}
}

func TestAnEmptyPathIsAcceptedAsUnset(t *testing.T) {
	dir := t.TempDir()
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if !s.commitSteamPath("  ") {
		t.Fatal("an empty path was refused")
	}
	cfg, readable, err := config.Load(dir, &rt.Fake{})
	if err != nil || !readable {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
	if cfg.SteamPath != "" {
		t.Fatalf("saved path = %q, want it unset", cfg.SteamPath)
	}
	if facts := s.snapshot(); facts.SteamPathValid || facts.Deployment != app.NotApplied {
		t.Fatalf("facts = %+v, want an unset path", facts)
	}
}

func TestTheLanguageAppliesAtOnceAndIsRemembered(t *testing.T) {
	dir := t.TempDir()
	host := &rt.Fake{}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if !slices.Contains(shown(t, s), "未应用") {
		t.Fatalf("texts = %q, want the Chinese screen", shown(t, s))
	}
	s.actions.SetLanguage(app.LanguageEnglish)
	if !slices.Contains(shown(t, s), "Not applied") {
		t.Fatalf("texts = %q, want the English screen", shown(t, s))
	}

	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
	if cfg.Language != app.LanguageEnglish {
		t.Fatalf("saved language = %d, want the explicit choice", cfg.Language)
	}
	restarted := newSession(host, noSleep, "zh-CN", cfg, readable, t.TempDir())
	if !slices.Contains(shown(t, restarted), "Not applied") {
		t.Fatalf("texts = %q, the explicit choice must beat the Chinese locale", shown(t, restarted))
	}
}

func TestAnUnreadableConfigOpensTheWizardOnItsFirstStep(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{}, false, t.TempDir())
	texts := shown(t, s)
	if !slices.Contains(texts, "首次运行向导") || !slices.Contains(texts, "第 1 / 4 步") {
		t.Fatalf("texts = %q, want the wizard on the language step", texts)
	}
	if slices.Contains(texts, "未应用") {
		t.Fatalf("texts = %q, the main screen must wait for the wizard", texts)
	}
}

func TestThePathStepKeepsTheWizardUntilTheDownloadStep(t *testing.T) {
	dir := t.TempDir()
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newSession(host, noSleep, "zh-CN", config.Config{}, false, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if !slices.Contains(shown(t, s), "首次运行向导") {
		t.Fatalf("texts = %q, want the wizard first", shown(t, s))
	}
	s.actions.WizardNext()
	s.actions.WizardNext()
	if !slices.Contains(shown(t, s), "第 3 / 4 步") {
		t.Fatalf("texts = %q, want the path step", shown(t, s))
	}
	s.uiState.PathDraft = steam
	s.actions.WizardNext()

	texts := shown(t, s)
	if !slices.Contains(texts, "第 4 / 4 步") {
		t.Fatalf("texts = %q, the valid path must reach the download step", texts)
	}
	if slices.Contains(texts, "未应用") {
		t.Fatalf("texts = %q, the main screen must wait for the wizard", texts)
	}
	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
	if cfg.SteamPath != steam {
		t.Fatalf("saved path = %q, want the committed one", cfg.SteamPath)
	}
}

func TestAnInvalidPathStaysOnThePathStep(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{}, false, t.TempDir())
	s.actions.WizardNext()
	s.actions.WizardNext()
	s.uiState.PathDraft = "Z:/not-a-directory"
	s.actions.WizardNext()

	if s.uiState.PathError != true {
		t.Error("the invalid path was not marked")
	}
	if !slices.Contains(shown(t, s), "第 3 / 4 步") {
		t.Fatalf("texts = %q, the wizard must stay on the path step", shown(t, s))
	}
}

func TestBrowseCommitsThePickedDirectoryOnTheUIThread(t *testing.T) {
	dir := t.TempDir()
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}, Picked: steam}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	hops := 0
	s.onUI = func(fn func()) {
		hops++
		fn()
	}

	s.pickSteamPath("")

	if hops != 1 {
		t.Fatalf("the pick hopped to the UI thread %d times, want once", hops)
	}
	if host.PickStart != "" {
		t.Fatalf("the dialog started at %q, want the unset path", host.PickStart)
	}
	if facts := s.snapshot(); facts.SteamPath != steam || !facts.SteamPathValid {
		t.Fatalf("facts = %+v, want the picked path", facts)
	}
	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable || cfg.SteamPath != steam {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
}

func TestARepeatedCommitDoesNotWriteAgain(t *testing.T) {
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newTestSession(t, host, "")
	writes := 0
	s.persist = func(config.Config) error {
		writes++
		return nil
	}

	if !s.actions.CommitPath(steam) {
		t.Fatal("a readable directory was refused")
	}
	if !s.actions.CommitPath(" " + steam + " ") {
		t.Fatal("the same directory, trimmed, was refused")
	}
	if writes != 1 {
		t.Fatalf("the config was written %d times, want once", writes)
	}
}

func TestAStaleSavedPathOpensTheWizardAskingForANewOne(t *testing.T) {
	steamRegistry = func(string, string) string { return "" }
	t.Cleanup(func() { steamRegistry = registryString })

	dir := t.TempDir()
	gone := filepath.Join(t.TempDir(), "gone")
	if err := config.Save(dir, config.Config{SteamPath: gone, Language: app.LanguageChinese}); err != nil {
		t.Fatal(err)
	}

	cfg, readable, err := config.Load(dir, &rt.Fake{})
	if err == nil {
		t.Fatal("a saved path that is not a directory came back without a reason")
	}
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", cfg, readable, t.TempDir())
	if !slices.Contains(shown(t, s), "首次运行向导") {
		t.Fatalf("texts = %q, want the wizard", shown(t, s))
	}
	if s.uiState.PathDraft != "" {
		t.Fatalf("draft = %q, want the stale path dropped for the path step", s.uiState.PathDraft)
	}
	if !s.uiState.PathNotFound {
		t.Fatal("a stale saved path shows no hint for the empty field")
	}
}

func TestSwitchingTheEntryRedrivesTheDeploymentWithoutADeploy(t *testing.T) {
	host, root := steamHost(t, false)
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "version.dll"} {
		host.Files[filepath.Join(root, name)] = true
	}
	cfg := config.Config{SteamPath: root, Language: app.LanguageAuto, Source: source.Community(), Entry: source.EntryEarly}
	s := newSession(host, noSleep, "zh-CN", cfg, true, t.TempDir())
	dir := t.TempDir()
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if got := s.factsWithoutGate().Deployment; got != app.Applied {
		t.Fatalf("the early entry reads %v, want applied", got)
	}
	s.setEntry(source.EntryDefault)
	if got := s.factsWithoutGate().Deployment; got != app.NotApplied {
		t.Errorf("the default entry reads %v, want not applied", got)
	}
	if len(host.Spawned) != 0 {
		t.Errorf("switching the entry spawned %d processes", len(host.Spawned))
	}

	loaded, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("Load: %v, readable %v", err, readable)
	}
	if loaded.Entry != source.EntryDefault || loaded.Source.Key != source.KeyCommunity {
		t.Errorf("saved %+v with entry %v, want the community source with the default entry", loaded.Source, loaded.Entry)
	}
}

func TestApplyingASourceSavesItAndRedrivesTheLayout(t *testing.T) {
	host, root := steamHost(t, false)
	for _, name := range []string{"SteamDaddy.dll", "xinput1_4.dll", "version.dll"} {
		host.Files[filepath.Join(root, name)] = true
	}
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto}, true, t.TempDir())
	dir := t.TempDir()
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if got := s.factsWithoutGate().Deployment; got != app.NotApplied {
		t.Fatalf("the community source reads %v, want not applied", got)
	}
	applied, ok := s.applySource(source.Draft{Builtin: source.KeySteamDaddy})
	if !ok || applied.Key != source.KeySteamDaddy {
		t.Fatalf("the draft applied to %+v (%v), want the SteamDaddy source", applied, ok)
	}
	if got := s.factsWithoutGate().Deployment; got != app.Applied {
		t.Errorf("the SteamDaddy layout reads %v, want applied", got)
	}
	if len(host.Spawned) != 0 {
		t.Errorf("applying a source spawned %d processes", len(host.Spawned))
	}

	loaded, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("Load: %v, readable %v", err, readable)
	}
	if loaded.Source.Key != source.KeySteamDaddy || loaded.Source.Layout != source.LayoutSteamDaddy {
		t.Errorf("saved source %+v, want the SteamDaddy source", loaded.Source)
	}
}

func TestAnEntryTheNewSourceCannotTakeFallsBackToTheDefault(t *testing.T) {
	host, root := steamHost(t, false)
	cfg := config.Config{SteamPath: root, Language: app.LanguageAuto, Source: source.Community(), Entry: source.EntryEarly}
	s := newSession(host, noSleep, "zh-CN", cfg, true, t.TempDir())
	dir := t.TempDir()
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if _, ok := s.applySource(source.Draft{Builtin: source.KeyOfficial}); !ok {
		t.Fatal("the official source was refused")
	}
	loaded, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("Load: %v, readable %v", err, readable)
	}
	if loaded.Source.Key != source.KeyOfficial || loaded.Entry != source.EntryDefault {
		t.Errorf("saved %+v with entry %v, want the official source with the default entry", loaded.Source, loaded.Entry)
	}
}

func TestARejectedDraftLeavesTheSavedSourceAlone(t *testing.T) {
	host, root := steamHost(t, false)
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto}, true, t.TempDir())
	saved := 0
	s.persist = func(config.Config) error { saved++; return nil }

	if _, ok := s.applySource(source.Draft{Custom: "github.com/someone/patch"}); ok {
		t.Error("an invalid address was applied")
	}
	if saved != 0 {
		t.Errorf("an invalid address was saved %d times", saved)
	}
}

func TestClosingTheDialogDropsTheSourceDraft(t *testing.T) {
	host, root := steamHost(t, false)
	s := newTestSession(t, host, root)

	s.uiState.SettingsOpen = true
	s.uiState.SourceChoice = source.KeySteamDaddy
	s.uiState.SourceCustom = "https://github.com/someone/patch"
	ui.NewTester(s.view, 480, 360)

	if s.uiState.SourceChoice != source.KeySteamDaddy || s.uiState.SourceCustom == "" {
		t.Fatalf("an open dialog lost the draft: %q %q", s.uiState.SourceChoice, s.uiState.SourceCustom)
	}

	s.uiState.SettingsOpen = false
	ui.NewTester(s.view, 480, 360)

	if s.uiState.SourceChoice != source.KeyCommunity || s.uiState.SourceCustom != "" {
		t.Errorf("a closed dialog kept the draft: %q %q", s.uiState.SourceChoice, s.uiState.SourceCustom)
	}
	if s.uiState.Source.Key != source.KeyCommunity {
		t.Errorf("the saved source reads %q, want the community one", s.uiState.Source.Key)
	}
}

func TestACustomAddressIsSavedAndShownAgain(t *testing.T) {
	host, root := steamHost(t, false)
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto}, true, t.TempDir())
	dir := t.TempDir()
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	const address = "https://github.com/someone/patch"

	applied, ok := s.applySource(source.Draft{Custom: address})
	if !ok || applied.Key != address || !applied.Custom {
		t.Fatalf("the draft applied to %+v (%v), want the custom address", applied, ok)
	}
	loaded, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("Load: %v, readable %v", err, readable)
	}
	if loaded.Source.Key != address {
		t.Errorf("the saved source is %q, want %q", loaded.Source.Key, address)
	}

	shown(t, s)
	if s.uiState.SourceCustom != address {
		t.Errorf("the address field reads %q, want %q", s.uiState.SourceCustom, address)
	}
}

func zipped(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range files {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func releaseBody(version string, assets ...string) []byte {
	var listed []string
	for i, url := range assets {
		listed = append(listed, fmt.Sprintf(`{"name": "asset-%d.zip", "browser_download_url": %q}`, i, url))
	}
	return []byte(fmt.Sprintf(`{"tag_name": %q, "assets": [%s]}`, version, strings.Join(listed, ", ")))
}

func communityURL() string {
	return patch.LatestURL(source.Community().Repository)
}

func runCommand(t *testing.T, s *session, run func() bool) {
	t.Helper()
	drawn := make(chan struct{}, 8)
	s.frame = func() { drawn <- struct{}{} }
	if !run() {
		t.Fatal("the command was refused")
	}
	select {
	case <-drawn:
	case <-time.After(5 * time.Second):
		t.Fatal("the command never started")
	}
	select {
	case <-drawn:
	case <-time.After(5 * time.Second):
		t.Fatal("the command never finished")
	}
}

func TestTheCheckReportsANewPatchWithoutDownloadingAnything(t *testing.T) {
	host := &rt.Fake{Responses: map[string][]byte{communityURL(): releaseBody("v1.2.0", "https://example.test/patch-release.zip")}}
	s := newTestSession(t, host, "")
	runCommand(t, s, s.checkUpdate)

	facts := s.factsWithoutGate()
	if !facts.UpdateChecked || facts.UpdateFailed || facts.OnlineVersion != "v1.2.0" {
		t.Fatalf("the check landed in %+v", facts)
	}
	if !slices.Contains(shown(t, s), "发现新补丁") || !slices.Contains(shown(t, s), "下载并解压新版本") {
		t.Errorf("texts %q, want the new patch conclusion and the download", shown(t, s))
	}
	if _, err := os.Stat(patch.Dir(s.dir, source.Community())); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the check wrote into the cache: %v", err)
	}
}

func TestTheDownloadFillsThePerSourceCacheAndSettlesOnUpToDate(t *testing.T) {
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "steam.exe"))
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"xinput1_4.dll":     "loader",
		"dwmapi.dll":        "entry",
		"OpenSteamTool.lib": "artifact",
		"notes.md":          "unlisted",
	})
	host := &rt.Fake{
		Responses: map[string][]byte{
			communityURL():                           releaseBody("v1.2.0", "https://example.test/patch-release.zip"),
			"https://example.test/patch-release.zip": archive,
		},
	}
	s := newTestSession(t, host, steam)
	runCommand(t, s, s.checkUpdate)
	runCommand(t, s, s.downloadPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.Downloaded {
		t.Errorf("the download result is %v, want a finished download", facts.Result)
	}
	dir := patch.Dir(s.dir, source.Community())
	cached, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the cache: %v", err)
	}
	var names []string
	for _, entry := range cached {
		names = append(names, entry.Name())
	}
	slices.Sort(names)
	want := []string{"OpenSteamTool.dll", "dwmapi.dll", "version.txt", "xinput1_4.dll"}
	if !slices.Equal(names, want) {
		t.Errorf("the cache holds %q, want %q", names, want)
	}
	record, err := os.ReadFile(filepath.Join(dir, "version.txt"))
	if err != nil || strings.TrimSpace(string(record)) != "v1.2.0" {
		t.Errorf("the version record is %q (%v), want v1.2.0", record, err)
	}
	if !slices.Contains(shown(t, s), "补丁已是最新") {
		t.Errorf("texts %q, want the up to date conclusion", shown(t, s))
	}
	if _, err := os.Stat(filepath.Join(steam, "OpenSteamTool.dll")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the download deployed into the Steam directory: %v", err)
	}
	if s.config().Snapshot != (config.DeploymentSnapshot{}) {
		t.Errorf("the download wrote the snapshot %+v", s.config().Snapshot)
	}
}

func TestARejectedPackageLeavesTheCacheByteIdentical(t *testing.T) {
	host := &rt.Fake{
		Responses: map[string][]byte{
			communityURL():                           releaseBody("v2.0.0", "https://example.test/patch-release.zip"),
			"https://example.test/patch-release.zip": zipped(t, map[string]string{"OpenSteamTool.dll": "payload", "xinput1_4.dll": "loader"}),
		},
	}
	s := newTestSession(t, host, "")
	if err := patch.Install(patch.Dir(s.dir, source.Community()), []patch.File{{Name: "OpenSteamTool.dll", Data: []byte("old payload")}}, "v1"); err != nil {
		t.Fatal(err)
	}
	runCommand(t, s, s.checkUpdate)
	runCommand(t, s, s.downloadPatch)

	if facts := s.factsWithoutGate(); facts.Result != app.MissingMember {
		t.Errorf("the rejected download reports %v, want the missing member", facts.Result)
	}
	data, err := os.ReadFile(filepath.Join(patch.Dir(s.dir, source.Community()), "OpenSteamTool.dll"))
	if err != nil || string(data) != "old payload" {
		t.Errorf("the cache payload reads %q (%v), want the old one", data, err)
	}
}

func TestABrokenPackageReportsALocalFileFailure(t *testing.T) {
	host := &rt.Fake{
		Responses: map[string][]byte{
			communityURL():                           releaseBody("v2.0.0", "https://example.test/patch-release.zip"),
			"https://example.test/patch-release.zip": []byte("<html>not a zip</html>"),
		},
	}
	s := newTestSession(t, host, "")
	runCommand(t, s, s.checkUpdate)
	runCommand(t, s, s.downloadPatch)

	if facts := s.factsWithoutGate(); facts.Result != app.LocalIOFailed {
		t.Errorf("the broken package reports %v, want the local file failure", facts.Result)
	}
	if _, err := os.Stat(patch.Dir(s.dir, source.Community())); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the friendly cache was written anyway: %v", err)
	}
}

func TestAnOfflineCheckReportsAFailureAndNeverReadsAsCurrent(t *testing.T) {
	host := &rt.Fake{FetchError: errors.New("no route to host")}
	s := newTestSession(t, host, "")
	runCommand(t, s, s.checkUpdate)

	facts := s.factsWithoutGate()
	if !facts.UpdateFailed || facts.UpdateChecked {
		t.Fatalf("the offline check landed in %+v", facts)
	}
	texts := shown(t, s)
	if !slices.Contains(texts, "检查失败") {
		t.Errorf("texts %q, want the failed conclusion", texts)
	}
	if slices.Contains(texts, "补丁已是最新") {
		t.Errorf("texts %q claim the patch is current", texts)
	}
}

func TestTheDownloadWithoutACheckIsRefused(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, "")
	if s.downloadPatch() {
		t.Error("a download without a check was taken")
	}
	if s.snapshot().Busy != app.Idle {
		t.Error("the refused download left the gate busy")
	}
}

func TestASourceChangeForgetsThePreviousCheck(t *testing.T) {
	host := &rt.Fake{Responses: map[string][]byte{communityURL(): releaseBody("v1.2.0", "https://example.test/patch-release.zip")}}
	s := newTestSession(t, host, "")
	runCommand(t, s, s.checkUpdate)
	if !s.factsWithoutGate().UpdateChecked {
		t.Fatal("the check did not land")
	}

	if _, ok := s.applySource(source.Draft{Builtin: source.KeySteamDaddy}); !ok {
		t.Fatal("the source change was rejected")
	}
	facts := s.factsWithoutGate()
	if facts.UpdateChecked || facts.OnlineVersion != "" {
		t.Errorf("the previous source's check survived: %+v", facts)
	}
	if s.downloadPatch() {
		t.Error("a download was offered from the previous source's result")
	}
}

func patchedSteam(t *testing.T, running bool) (*rt.Fake, *session, string) {
	t.Helper()
	host, root := steamHost(t, running)
	host.OnSpawn = func(call rt.Launch) {
		if slices.Equal(call.Args, []string{"-shutdown"}) {
			host.Procs = nil
			return
		}
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}
	host.Responses = map[string][]byte{
		communityURL(): releaseBody("v1.2.0", "https://example.test/patch-release.zip"),
		"https://example.test/patch-release.zip": zipped(t, map[string]string{
			"OpenSteamTool.dll": "payload",
			"xinput1_4.dll":     "loader",
			"dwmapi.dll":        "entry",
		}),
	}
	s := newTestSession(t, host, root)
	runCommand(t, s, s.checkUpdate)
	runCommand(t, s, s.downloadPatch)
	return host, s, root
}

func TestApplyingThePatchDeploysLaunchesAndSnapshots(t *testing.T) {
	host, s, root := patchedSteam(t, true)
	manifest := filepath.Join(root, "package", "steam_client_win32.manifest")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte("\"version\" \"1712345678\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hidden := 0
	s.hide = func() { hidden++ }

	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.PatchApplied || facts.Busy != app.Idle || facts.PatchFailed || facts.Conflict != "" {
		t.Fatalf("facts = %+v, want an applied patch with an idle gate", facts)
	}
	if facts.Deployment != app.Applied || !facts.SteamRunning {
		t.Errorf("facts = %+v, want the deployed patch and a running Steam", facts)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s did not land in the Steam directory", name)
		}
	}
	if len(host.Spawned) != 2 || !slices.Equal(host.Spawned[0].Args, []string{"-shutdown"}) || len(host.Spawned[1].Args) != 0 {
		t.Fatalf("spawned %+v, want the shutdown signal then a bare start", host.Spawned)
	}
	want := config.DeploymentSnapshot{Source: source.KeyCommunity, Layout: source.LayoutOpenSteamTool, ClientBuild: "1712345678", PatchVersion: "v1.2.0"}
	if s.config().Snapshot != want {
		t.Errorf("the snapshot is %+v, want %+v", s.config().Snapshot, want)
	}
	if hidden != 1 {
		t.Errorf("hid the window %d times, want once for the running Steam", hidden)
	}
	if !slices.Contains(shown(t, s), "补丁已应用") {
		t.Errorf("texts = %q, want the applied result", shown(t, s))
	}
}

func TestApplyingWithoutASteamPathTouchesNothing(t *testing.T) {
	host := &rt.Fake{}
	s := newTestSession(t, host, "")

	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.PathUnset || !facts.PatchFailed {
		t.Fatalf("facts = %+v, want a refused apply with the failure flag", facts)
	}
	if len(host.Writes) != 0 || len(host.Mkdirs) != 0 || len(host.Spawned) != 0 {
		t.Errorf("a refused apply wrote %v, made %v and spawned %v", host.Writes, host.Mkdirs, host.Spawned)
	}
}

func TestApplyingWithoutADownloadedPatchTouchesNothing(t *testing.T) {
	host, root := steamHost(t, true)
	s := newTestSession(t, host, root)

	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.MissingMember || !facts.PatchFailed {
		t.Fatalf("facts = %+v, want the missing member with the failure flag", facts)
	}
	if len(host.Spawned) != 0 || len(host.Writes) != 0 {
		t.Errorf("a refused apply spawned %v and wrote %v", host.Spawned, host.Writes)
	}
}

func TestApplyingWithoutSteamProgramStopsNothing(t *testing.T) {
	host, s, root := patchedSteam(t, true)
	delete(host.Files, filepath.Join(root, "steam.exe"))

	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.MissingSteamProgram || !facts.PatchFailed {
		t.Fatalf("facts = %+v, want the missing program with the failure flag", facts)
	}
	if len(host.Spawned) != 0 {
		t.Errorf("spawned %+v, want Steam left alone", host.Spawned)
	}
}

func TestAConflictingFileBlocksThenTheRemovalDeploys(t *testing.T) {
	host, s, root := patchedSteam(t, false)
	stale := filepath.Join(root, "version.dll")
	host.Files[stale] = true

	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Conflict != "version.dll" || !facts.PatchFailed || facts.Result != app.DeploymentFailed {
		t.Fatalf("facts = %+v, want a blocked deploy naming the conflicting file", facts)
	}
	if !host.IsFile(stale) || host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the blocked deploy changed the Steam directory")
	}
	if !slices.Contains(shown(t, s), "检测到 version.dll，需先移除") || !slices.Contains(shown(t, s), "移除并部署") {
		t.Errorf("texts = %q, want the conflict and its removal", shown(t, s))
	}

	runCommand(t, s, s.removeConflictAndDeploy)

	facts = s.factsWithoutGate()
	if facts.Result != app.PatchApplied || facts.PatchFailed || facts.Conflict != "" {
		t.Fatalf("facts = %+v, want the retried deploy applied", facts)
	}
	if host.IsFile(stale) || !host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the retry did not replace the conflicting file with the entry")
	}
}

func TestAFailedLaunchKeepsTheSnapshotAndTheFailuresApart(t *testing.T) {
	host, s, root := patchedSteam(t, false)
	host.OnSpawn = func(rt.Launch) {}
	hidden := 0
	s.hide = func() { hidden++ }
	runCommand(t, s, s.applyPatch)

	facts := s.factsWithoutGate()
	if facts.Result != app.StartSteamFailed || !facts.PatchFailed {
		t.Fatalf("facts = %+v, want the failed launch with the failure flag", facts)
	}
	if s.config().Snapshot != (config.DeploymentSnapshot{}) {
		t.Errorf("a failed launch wrote the snapshot %+v", s.config().Snapshot)
	}
	if !host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the deploy was rolled back by the failed launch")
	}
	if hidden != 0 {
		t.Errorf("hid the window %d times for a failed launch", hidden)
	}
}

func TestAStateRefreshClearsTheApplyFailure(t *testing.T) {
	host, s, root := patchedSteam(t, false)
	host.Files[filepath.Join(root, "version.dll")] = true

	runCommand(t, s, s.applyPatch)
	if facts := s.factsWithoutGate(); !facts.PatchFailed || facts.Conflict != "version.dll" {
		t.Fatalf("facts = %+v, want the blocked deploy", facts)
	}

	s.setLanguage(app.LanguageEnglish)

	facts := s.factsWithoutGate()
	if facts.PatchFailed || facts.Conflict != "" {
		t.Errorf("facts = %+v, the refresh kept the failure", facts)
	}
}

func appliedSteam(t *testing.T, running bool) (*rt.Fake, *session, string) {
	t.Helper()
	host, root := steamHost(t, running)
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		host.Files[filepath.Join(root, name)] = true
	}
	return host, newTestSession(t, host, root), root
}

func TestAnUninstallWithoutASteamRunningRemovesOnlyTheLayoutFiles(t *testing.T) {
	host, s, root := appliedSteam(t, false)
	lua := filepath.Join(root, "config", "lua")
	host.Dirs[lua] = true
	host.Files[filepath.Join(lua, "unlock.lua")] = true
	host.Files[filepath.Join(root, "user.lua")] = true

	runCommand(t, s, s.uninstall)

	facts := s.factsWithoutGate()
	if facts.Result != app.PatchRemoved || facts.Deployment != app.NotApplied || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the patch removed with an idle gate", facts)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		if host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s survived the removal", name)
		}
	}
	for _, path := range []string{filepath.Join(root, "user.lua"), filepath.Join(lua, "unlock.lua")} {
		if !host.IsFile(path) {
			t.Errorf("%s was removed, want the user's .lua files kept", path)
		}
	}
	if !host.IsDir(lua) {
		t.Error("the Lua directory was removed")
	}
	if len(host.Spawned) != 0 || len(host.Killed) != 0 {
		t.Errorf("spawned %v and killed %v, the removal must leave Steam alone", host.Spawned, host.Killed)
	}
	if len(host.Removes) != 3 {
		t.Errorf("removed %v, want only the layout's files", host.Removes)
	}
	if slices.Contains(shown(t, s), "确认") {
		t.Error("the removal asked for confirmation with Steam stopped")
	}

	runCommand(t, s, s.uninstall)
	if facts := s.factsWithoutGate(); facts.Result != app.PatchRemoved {
		t.Errorf("facts = %+v, want a repeated removal to succeed", facts)
	}
}

func TestARunningRemovalAsksFirstAndADeclineChangesNothing(t *testing.T) {
	host, s, root := appliedSteam(t, true)

	if !s.exitSteamAndUninstall() {
		t.Fatal("the request was refused")
	}
	facts := s.factsWithoutGate()
	if facts.Pending != app.ConfirmExitRemove || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the question pending with an idle gate", facts)
	}
	if !slices.Contains(shown(t, s), "Steam 正在运行。是否自动关闭 Steam 并继续？") {
		t.Errorf("texts = %q, want the question", shown(t, s))
	}
	if len(host.Removes) != 0 || len(host.Spawned) != 0 {
		t.Errorf("the question changed the Steam directory: removed %v, spawned %v", host.Removes, host.Spawned)
	}

	s.declinePending()

	facts = s.factsWithoutGate()
	if facts.Pending != app.NoConfirmation || facts.Result != app.NoResult || facts.Deployment != app.Applied || !facts.SteamRunning {
		t.Fatalf("facts = %+v, want the decline to change nothing", facts)
	}
	if !host.IsFile(filepath.Join(root, "dwmapi.dll")) || len(host.Removes) != 0 {
		t.Errorf("the decline removed %v", host.Removes)
	}
	if slices.Contains(shown(t, s), "确认") {
		t.Error("the question survived the decline")
	}
}

func TestAConfirmedExitAndRemovalClosesSteamThenRemovesThePatch(t *testing.T) {
	host, s, root := appliedSteam(t, true)
	host.OnSpawn = func(call rt.Launch) {
		if slices.Equal(call.Args, []string{"-shutdown"}) {
			host.Procs = nil
			return
		}
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}
	s.exitSteamAndUninstall()

	runCommand(t, s, func() bool { s.confirmPending(); return true })

	facts := s.factsWithoutGate()
	if facts.Result != app.PatchRemoved || facts.Deployment != app.NotApplied || facts.SteamRunning || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the patch removed and Steam left closed", facts)
	}
	if len(host.Spawned) != 1 || !slices.Equal(host.Spawned[0].Args, []string{"-shutdown"}) {
		t.Fatalf("spawned %+v, want only the graceful shutdown signal", host.Spawned)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		if host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s survived the removal", name)
		}
	}
}

func TestACloseFailureLeavesThePatchAndTheAppliedState(t *testing.T) {
	host, s, root := appliedSteam(t, true)
	host.OnSpawn = func(rt.Launch) {}
	host.OnTerminate = func(int) {}
	s.exitSteamAndUninstall()

	runCommand(t, s, func() bool { s.confirmPending(); return true })

	facts := s.factsWithoutGate()
	if facts.Result != app.CloseSteamFailed || facts.Deployment != app.Applied || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the failed close with the patch still applied", facts)
	}
	if len(host.Removes) != 0 {
		t.Errorf("the failed close removed %v", host.Removes)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s was removed after the failed close", name)
		}
	}
	if !slices.Contains(shown(t, s), "关闭 Steam 失败") {
		t.Errorf("texts = %q, want the failed close in the status bar", shown(t, s))
	}
}

func TestAnUninstallAndRestartLeavesSteamRunningAndThePatchGone(t *testing.T) {
	host, s, root := appliedSteam(t, false)
	host.OnSpawn = func(call rt.Launch) {
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}

	runCommand(t, s, s.uninstallAndRestart)

	facts := s.factsWithoutGate()
	if facts.Result != app.PatchRemoved || facts.Deployment != app.NotApplied || !facts.SteamRunning || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the patch removed and Steam running again", facts)
	}
	if len(host.Spawned) != 1 || len(host.Spawned[0].Args) != 0 {
		t.Fatalf("spawned %+v, want one bare start", host.Spawned)
	}
	if host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the entry survived the removal")
	}
	if slices.Contains(shown(t, s), "确认") {
		t.Error("the removal asked for confirmation with Steam stopped")
	}
}

func TestAnUninstallWithoutASteamPathTouchesNothing(t *testing.T) {
	host := &rt.Fake{}
	s := newTestSession(t, host, "")

	runCommand(t, s, s.uninstall)

	facts := s.factsWithoutGate()
	if facts.Result != app.PathUnset || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want a refused removal", facts)
	}
	if len(host.Removes) != 0 || len(host.Spawned) != 0 {
		t.Errorf("a refused removal removed %v and spawned %v", host.Removes, host.Spawned)
	}
}

func TestAnUninstallRequestWhileBusyIsDropped(t *testing.T) {
	host, s, _ := appliedSteam(t, true)
	if !s.take(app.BusyApplyPatch) {
		t.Fatal("the gate was not free")
	}

	if s.exitSteamAndUninstall() || s.uninstallAndRestart() {
		t.Fatal("a removal request was accepted while the gate was taken")
	}

	facts := s.factsWithoutGate()
	if facts.Pending != app.NoConfirmation || len(host.Removes) != 0 || len(host.Spawned) != 0 {
		t.Fatalf("facts = %+v, want the request dropped without a question", facts)
	}
}

func TestTheRunningRemovalAndRestartAsksBeforeItClosesSteam(t *testing.T) {
	host, s, root := appliedSteam(t, true)
	host.OnSpawn = func(call rt.Launch) {
		if slices.Equal(call.Args, []string{"-shutdown"}) {
			host.Procs = nil
			return
		}
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}

	if !s.uninstallAndRestart() {
		t.Fatal("the request was refused")
	}
	facts := s.factsWithoutGate()
	if facts.Pending != app.ConfirmRemoveAndRestart || len(host.Removes) != 0 || len(host.Spawned) != 0 {
		t.Fatalf("facts = %+v, want the question pending and nothing touched", facts)
	}

	runCommand(t, s, func() bool { s.confirmPending(); return true })

	facts = s.factsWithoutGate()
	if facts.Result != app.PatchRemoved || facts.Deployment != app.NotApplied || !facts.SteamRunning || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the patch removed and Steam restarted", facts)
	}
	if len(host.Spawned) != 2 || !slices.Equal(host.Spawned[0].Args, []string{"-shutdown"}) || len(host.Spawned[1].Args) != 0 {
		t.Fatalf("spawned %+v, want the shutdown signal then a bare start", host.Spawned)
	}
	if host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the entry survived the removal")
	}
}

func TestAFailedRestartAfterTheRemovalIsRecorded(t *testing.T) {
	host, s, root := appliedSteam(t, false)
	host.OnSpawn = func(rt.Launch) {}

	runCommand(t, s, s.uninstallAndRestart)

	facts := s.factsWithoutGate()
	if facts.Result != app.StartSteamFailed || facts.Deployment != app.NotApplied || facts.SteamRunning || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the removed patch and a failed start", facts)
	}
	if host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the entry survived the removal")
	}
	if !slices.Contains(shown(t, s), "启动 Steam 失败") {
		t.Errorf("texts = %q, want the failed start in the status bar", shown(t, s))
	}
}

type lockedHost struct {
	*rt.Fake
}

func (h lockedHost) Remove(string) error {
	return errors.New("the file is open in another process")
}

func TestAnUndeletableFileReportsTheRemovalFailure(t *testing.T) {
	host, root := steamHost(t, false)
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		host.Files[filepath.Join(root, name)] = true
	}
	s := newTestSession(t, lockedHost{host}, root)

	runCommand(t, s, s.uninstall)

	facts := s.factsWithoutGate()
	if facts.Result != app.RemovalFailed || facts.Busy != app.Idle {
		t.Fatalf("facts = %+v, want the failed removal with an idle gate", facts)
	}
	if facts.Deployment != app.Applied || facts.PatchFailed {
		t.Errorf("facts = %+v, want the untouched patch and no apply failure", facts)
	}
	if !slices.Contains(shown(t, s), "卸载失败") {
		t.Errorf("texts = %q, want the failed removal in the status bar", shown(t, s))
	}
}

func steamDaddySource() source.Source {
	owner, _ := source.BuiltinFor(source.LayoutSteamDaddy)
	return owner
}

func steamDaddySteam(t *testing.T, running bool) (*rt.Fake, *session, string) {
	t.Helper()
	host, root := steamHost(t, running)
	host.OnSpawn = func(call rt.Launch) {
		if slices.Equal(call.Args, []string{"-shutdown"}) {
			host.Procs = nil
			return
		}
		host.Procs = append(host.Procs, rt.Process{PID: call.PID, Exe: call.Exe})
	}
	daddy := steamDaddySource()
	packageURL := "https://example.test/daddy-release.zip"
	host.Responses = map[string][]byte{
		patch.LatestURL(daddy.Repository): releaseBody("v1.2.0", packageURL),
		packageURL: zipped(t, map[string]string{
			"SteamDaddy.dll":      "payload",
			"xinput1_4.dll":       "loader",
			"version.dll.disable": "entry",
			"cloud_redirect.dll":  "redirect",
			"steamdaddy.toml":     "default",
		}),
	}
	s := newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto, Source: daddy}, true, t.TempDir())
	runCommand(t, s, s.checkUpdate)
	runCommand(t, s, s.downloadPatch)
	return host, s, root
}

func TestAnEditedConfigIsArchivedOnRemovalAndRestoredOnDeploy(t *testing.T) {
	host, s, root := steamDaddySteam(t, false)
	runCommand(t, s, s.applyPatch)

	toml := filepath.Join(root, "steamdaddy.toml")
	if got := string(host.Contents[toml]); got != "default" {
		t.Fatalf("the deployed toml holds %q, want the published default", got)
	}
	host.Contents[toml] = []byte("edited")

	runCommand(t, s, s.uninstall)

	facts := s.factsWithoutGate()
	if facts.Result != app.PatchRemoved || facts.Deployment != app.NotApplied {
		t.Fatalf("facts = %+v, want the patch removed", facts)
	}
	archive := filepath.Join(s.dir, "userdata", patch.Slug(source.KeySteamDaddy), "steamdaddy.toml")
	if got := string(host.Contents[archive]); got != "edited" {
		t.Fatalf("the archive holds %q, want the edited config", got)
	}
	if host.IsFile(toml) {
		t.Error("the edited config survived the removal")
	}

	runCommand(t, s, s.applyPatch)

	if got := string(host.Contents[toml]); got != "edited" {
		t.Errorf("the deploy restored %q, want the edited config", got)
	}
}
