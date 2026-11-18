package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
)

func shown(t *testing.T, s *session) []string {
	t.Helper()
	return ui.NewTester(s.view, 480, 360).Texts()
}

func noSleep(time.Duration) {}

func newTestSession(host rt.Runtime, root string) *session {
	return newSession(host, noSleep, "zh-CN", config.Config{SteamPath: root, Language: app.LanguageAuto}, true)
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

	s := &session{facts: factsFor(app.LanguageChinese, "zh-CN", steam, rt.OS{})}
	if !slices.Contains(shown(t, s), "已应用") {
		t.Fatalf("texts = %q, want 已应用", shown(t, s))
	}
}

func TestARealSteamDirectoryWithoutTheLayoutReadsAsNotApplied(t *testing.T) {
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "steam.exe"))

	s := &session{facts: factsFor(app.LanguageChinese, "zh-CN", steam, rt.OS{})}
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
	s := newTestSession(host, root)
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
	s := newTestSession(host, root)
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
	s := newTestSession(host, root)
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
	s := newTestSession(host, root)
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
	s := newTestSession(host, root)

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

	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true)
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

	restarted := newSession(host, noSleep, "zh-CN", cfg, readable)
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
	s := newSession(rt.OS{}, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true)
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

	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true)
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
	s := newTestSession(&rt.Fake{}, t.TempDir())
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
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true)
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	var titles []string
	s.retitle = func(title string) { titles = append(titles, title) }

	if !slices.Contains(shown(t, s), "未应用") {
		t.Fatalf("texts = %q, want the Chinese screen", shown(t, s))
	}
	s.actions.SetLanguage(app.LanguageEnglish)
	if !slices.Contains(shown(t, s), "Not applied") {
		t.Fatalf("texts = %q, want the English screen", shown(t, s))
	}
	if !slices.Equal(titles, []string{"OpenSteamTool Manager"}) {
		t.Fatalf("titles = %q, want the window retitled once", titles)
	}

	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
	if cfg.Language != app.LanguageEnglish {
		t.Fatalf("saved language = %d, want the explicit choice", cfg.Language)
	}
	restarted := newSession(host, noSleep, "zh-CN", cfg, readable)
	if !slices.Contains(shown(t, restarted), "Not applied") {
		t.Fatalf("texts = %q, the explicit choice must beat the Chinese locale", shown(t, restarted))
	}
}

func TestAnUnreadableConfigAsksForSetup(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{}, false)
	texts := shown(t, s)
	if !slices.Contains(texts, "尚未完成设置") {
		t.Fatalf("texts = %q, want the setup notice", texts)
	}
	if !slices.Contains(texts, "打开设置") {
		t.Fatalf("texts = %q, want the way into the settings", texts)
	}
	if slices.Contains(texts, "未应用") {
		t.Fatalf("texts = %q, the main screen must wait for the setup", texts)
	}
}

func TestCommittingAPathLeavesTheSetupScreenBehind(t *testing.T) {
	dir := t.TempDir()
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}}
	s := newSession(host, noSleep, "zh-CN", config.Config{}, false)
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	if !slices.Contains(shown(t, s), "尚未完成设置") {
		t.Fatalf("texts = %q, want the setup screen first", shown(t, s))
	}
	if !s.actions.CommitPath(steam) {
		t.Fatal("a readable directory was refused")
	}

	texts := shown(t, s)
	if slices.Contains(texts, "尚未完成设置") {
		t.Fatalf("texts = %q, the setup screen must clear", texts)
	}
	if !slices.Contains(texts, "未应用") {
		t.Fatalf("texts = %q, want the main screen", texts)
	}
	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("load = %+v readable = %v err = %v", cfg, readable, err)
	}
	if cfg.SteamPath != steam {
		t.Fatalf("saved path = %q, want the committed one", cfg.SteamPath)
	}
}

func TestBrowseCommitsThePickedDirectoryOnTheUIThread(t *testing.T) {
	dir := t.TempDir()
	steam := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{steam: true}, Picked: steam}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true)
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
	s := newTestSession(host, "")
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

func TestAStaleSavedPathRoutesToSetupWithThePathKept(t *testing.T) {
	dir := t.TempDir()
	gone := filepath.Join(t.TempDir(), "gone")
	if err := config.Save(dir, config.Config{SteamPath: gone, Language: app.LanguageChinese}); err != nil {
		t.Fatal(err)
	}

	cfg, readable, err := config.Load(dir, &rt.Fake{})
	if err == nil {
		t.Fatal("a saved path that is not a directory came back without a reason")
	}
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", cfg, readable)
	if !slices.Contains(shown(t, s), "尚未完成设置") {
		t.Fatalf("texts = %q, want the setup screen", shown(t, s))
	}
	if s.uiState.PathDraft != gone {
		t.Fatalf("draft = %q, want the saved path for the Steam tab", s.uiState.PathDraft)
	}
}
