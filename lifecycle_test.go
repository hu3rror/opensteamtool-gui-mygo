package main

import (
	"errors"
	"github.com/egoist/mygo"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
	"opensteamtool-gui-mygo/internal/view"
)

func TestClosingTheWindowKeepsTheProcessInTheTray(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.tray = true
	visible := true
	var hides, balloons int
	var balloon string
	s.visible = func() bool { return visible }
	s.hide = func() {
		hides++
		visible = false
	}
	s.notify = func(body string) {
		balloons++
		balloon = body
	}

	if s.closeWindow() {
		t.Fatal("the close ended the process while the tray exists")
	}
	if hides != 1 || balloons != 1 {
		t.Fatalf("hides = %d, balloons = %d, want 1 and 1", hides, balloons)
	}
	if want := view.TrayRunning(s.resolvedLanguage()); balloon != want {
		t.Fatalf("balloon = %q, want %q", balloon, want)
	}

	visible = true
	s.closeWindow()
	if hides != 2 || balloons != 1 {
		t.Fatalf("hides = %d, balloons = %d, want 2 and 1", hides, balloons)
	}
}

func TestTheTrayNoticeShowsOnlyOncePerInstall(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.tray = true
	visible := true
	var balloons int
	s.visible = func() bool { return visible }
	s.hide = func() { visible = false }
	s.notify = func(string) { balloons++ }

	s.closeWindow()
	if balloons != 1 {
		t.Fatalf("balloons = %d, want the notice on the first hide", balloons)
	}
	cfg := s.config()
	if !cfg.TrayNoticeShown {
		t.Fatal("the notice was not recorded, it would show again after a restart")
	}

	restarted := newSession(&rt.Fake{}, noSleep, "zh-CN", cfg, true, t.TempDir())
	restarted.tray = true
	shown := true
	var again int
	restarted.visible = func() bool { return shown }
	restarted.hide = func() { shown = false }
	restarted.notify = func(string) { again++ }

	restarted.closeWindow()
	restarted.closeWindow()
	if again != 0 {
		t.Fatalf("balloons = %d after a restart, want none ever again", again)
	}
}

func TestClosingTheWindowWithoutATrayEndsTheProcess(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	var hides int
	s.hide = func() { hides++ }

	if !s.closeWindow() {
		t.Fatal("the close kept the process alive without a tray")
	}
	if hides != 0 {
		t.Fatalf("hides = %d, want 0", hides)
	}
}

func TestQuittingClosesTheWindowInsteadOfHidingIt(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.tray = true
	s.quitting = true
	var hides int
	s.hide = func() { hides++ }

	if !s.closeWindow() {
		t.Fatal("the close was prevented while quitting")
	}
	if hides != 0 {
		t.Fatalf("hides = %d, want 0", hides)
	}
}

func TestWakingShowsAndPaintsOnlyAHiddenWindow(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	visible := false
	var shows, frames int
	s.visible = func() bool { return visible }
	s.show = func() {
		shows++
		visible = true
	}
	s.frame = func() { frames++ }

	s.wake()
	if shows != 1 || frames != 1 {
		t.Fatalf("shows = %d, frames = %d, want 1 and 1", shows, frames)
	}

	s.wake()
	if shows != 1 || frames != 1 {
		t.Fatalf("shows = %d, frames = %d, want the visible window left alone", shows, frames)
	}
}

func TestTheTrayExitQuitsAtOnceWithoutAPatch(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	var quits int
	s.quit = func() { quits++ }

	s.requestExit()
	if quits != 1 {
		t.Fatalf("quits = %d, want the exit without a question", quits)
	}
	if s.facts.Pending != app.NoConfirmation {
		t.Fatalf("pending = %v, want none", s.facts.Pending)
	}
}

func TestTheTrayExitAsksForConfirmationThenQuits(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.facts.Deployment = app.Applied
	var quits int
	s.quit = func() { quits++ }

	s.requestExit()
	if s.facts.Pending != app.ConfirmExit {
		t.Fatalf("pending = %v, want %v", s.facts.Pending, app.ConfirmExit)
	}
	if quits != 0 {
		t.Fatalf("quits = %d, want 0 before the answer", quits)
	}

	s.confirmPending()
	if quits != 1 {
		t.Fatalf("quits = %d, want 1", quits)
	}
	if s.facts.Pending != app.NoConfirmation {
		t.Fatalf("pending = %v, want %v", s.facts.Pending, app.NoConfirmation)
	}
}

func TestTheExitWaitsForTheFileStepInFlight(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	var quits int
	s.quit = func() { quits++ }
	s.steps.Add(1)
	s.sleep = func(time.Duration) { s.steps.Add(-1) }

	s.exit()
	if quits != 1 {
		t.Fatalf("quits = %d, want 1", quits)
	}
	if s.steps.Load() != 0 {
		t.Fatalf("steps = %d, want the step in flight waited for", s.steps.Load())
	}
}

func TestTheExitGivesUpOnAFileStepThatNeverEnds(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	var quits int
	s.quit = func() { quits++ }
	s.steps.Add(1)
	s.sleep = func(time.Duration) {}

	s.exit()
	if quits != 1 {
		t.Fatalf("quits = %d, want the millisecond bound to end the process", quits)
	}
}

func TestTheSteamEdgesFollowTheVisibility(t *testing.T) {
	root := t.TempDir()
	host := &rt.Fake{}
	s := newTestSession(t, host, root)
	visible := true
	var hides, shows int
	s.visible = func() bool { return visible }
	s.hide = func() {
		hides++
		visible = false
	}
	s.show = func() {
		shows++
		visible = true
	}
	s.notify = func(string) {}

	s.poll()
	if hides != 0 || shows != 0 {
		t.Fatalf("hides = %d, shows = %d, want nothing while Steam is stopped", hides, shows)
	}

	host.Procs = []rt.Process{{PID: 100, Exe: filepath.Join(root, "steam.exe")}}
	s.poll()
	if hides != 1 || visible {
		t.Fatalf("hides = %d, visible = %v, want the running Steam to hide the window", hides, visible)
	}

	visible = true
	s.poll()
	if hides != 1 {
		t.Fatalf("hides = %d, want the reopened window kept", hides)
	}

	visible = false
	host.Procs = nil
	s.poll()
	if shows != 1 || !visible {
		t.Fatalf("shows = %d, visible = %v, want the stopped Steam to show the window", shows, visible)
	}
}

func TestAHiddenWindowRepaintsOnceAMinute(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	visible := false
	var frames int
	now := time.Unix(0, 0)
	s.visible = func() bool { return visible }
	s.frame = func() { frames++ }
	s.now = func() time.Time { return now }

	s.paint()
	if frames != 1 {
		t.Fatalf("frames = %d, want 1", frames)
	}

	now = now.Add(30 * time.Second)
	s.paint()
	if frames != 1 {
		t.Fatalf("frames = %d, want the hidden repaint left to once a minute", frames)
	}

	now = now.Add(30 * time.Second)
	s.paint()
	if frames != 2 {
		t.Fatalf("frames = %d, want 2", frames)
	}

	visible = true
	s.paint()
	if frames != 3 {
		t.Fatalf("frames = %d, want the shown window painted at once", frames)
	}
}

func TestOpeningTheLuaFolderCreatesAndOpensIt(t *testing.T) {
	root := t.TempDir()
	host := &rt.Fake{}
	s := newTestSession(t, host, root)
	s.facts.SteamPath = root
	s.facts.SteamPathValid = true

	s.openLuaFolder()

	want := filepath.Join(root, "config", "lua")
	if len(host.Mkdirs) != 1 || host.Mkdirs[0] != want {
		t.Fatalf("mkdirs = %q, want [%q]", host.Mkdirs, want)
	}
	if len(host.Opens) != 1 || host.Opens[0] != want {
		t.Fatalf("opens = %q, want [%q]", host.Opens, want)
	}
	if s.facts.Result == app.LocalIOFailed {
		t.Fatal("the open reported a local file failure")
	}
}

func TestAFailingLuaFolderReportsALocalFileFailure(t *testing.T) {
	root := t.TempDir()
	host := &rt.Fake{OpenError: errors.New("no handler")}
	s := newTestSession(t, host, root)
	s.facts.SteamPath = root
	s.facts.SteamPathValid = true

	s.openLuaFolder()

	if s.facts.Result != app.LocalIOFailed {
		t.Fatalf("result = %v, want %v", s.facts.Result, app.LocalIOFailed)
	}
	if texts := shown(t, s); !slices.Contains(texts, "本地文件操作失败") {
		t.Fatalf("texts = %q, want 本地文件操作失败", texts)
	}
}

func TestTheLuaFolderIsLeftAloneWithoutAValidSteamPath(t *testing.T) {
	host := &rt.Fake{}
	s := newTestSession(t, host, t.TempDir())
	s.facts.SteamPathValid = false

	s.openLuaFolder()

	if len(host.Mkdirs) != 0 || len(host.Opens) != 0 {
		t.Fatalf("mkdirs = %q, opens = %q, want nothing touched", host.Mkdirs, host.Opens)
	}
}

func TestTheLuaTabCarriesTheFolderOfTheSourceLayout(t *testing.T) {
	root := t.TempDir()
	lua := filepath.Join(root, "config", "lua")
	host := &rt.Fake{
		Dirs:  map[string]bool{root: true, lua: true},
		Files: map[string]bool{filepath.Join(lua, "manifest.lua"): true},
	}
	s := newTestSession(t, host, root)

	facts := s.snapshot()
	if facts.LuaRelative != "config/lua" || facts.LuaDir != lua || !facts.LuaManifest {
		t.Fatalf("lua = %q in %q (manifest %v), want the OpenSteamTool folder and its script",
			facts.LuaDir, facts.LuaRelative, facts.LuaManifest)
	}

	if _, ok := s.actions.ApplySource(source.Draft{Builtin: source.KeySteamDaddy}); !ok {
		t.Fatal("the SteamDaddy source was rejected")
	}
	facts = s.snapshot()
	if facts.LuaRelative != "config/stplug-in" || facts.LuaDir != filepath.Join(root, "config", "stplug-in") || facts.LuaManifest {
		t.Fatalf("lua = %q in %q (manifest %v), want the SteamDaddy folder without a script",
			facts.LuaDir, facts.LuaRelative, facts.LuaManifest)
	}
}

func TestTheLuaTabOpensTheFolderOfTheLayout(t *testing.T) {
	root := t.TempDir()
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	s := newTestSession(t, host, root)

	s.actions.OpenLua()

	want := filepath.Join(root, "config", "lua")
	if len(host.Opens) != 1 || host.Opens[0] != want {
		t.Fatalf("opens = %q, want [%q]", host.Opens, want)
	}
}

func TestTheLuaTabHasNoFolderWithoutAValidSteamPath(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())

	facts := s.snapshot()

	if facts.LuaDir != "" || facts.LuaRelative != "" || facts.LuaManifest {
		t.Fatalf("lua = %q in %q, want no folder without a valid Steam path", facts.LuaDir, facts.LuaRelative)
	}
}

func TestTheSteamCfgFactFollowsTheFile(t *testing.T) {
	for _, present := range []bool{false, true} {
		root := t.TempDir()
		files := map[string]bool{}
		if present {
			files[filepath.Join(root, "steam.cfg")] = true
		}
		host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: files}
		s := newTestSession(t, host, root)

		if got := s.snapshot().SteamCfg; got != present {
			t.Fatalf("steam.cfg = %v, want %v", got, present)
		}
	}
}

func TestTheTrayMenuListsTheFiveEntriesInOrder(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	lang := s.resolvedLanguage()
	update := &mygo.MenuItem{Label: "检查更新…"}
	want := []string{view.Show(lang), view.RestartSteam(lang), view.OpenLuaFolder(lang), update.Label, view.Exit(lang)}

	items := trayMenu(s, update).Items()
	if len(items) != len(want) {
		t.Fatalf("items = %d, want %d", len(items), len(want))
	}
	for i, label := range want {
		if items[i].Label != label {
			t.Fatalf("items[%d] = %q, want %q", i, items[i].Label, label)
		}
	}
	if !items[2].Disabled {
		t.Fatal("the Lua folder entry is enabled without a valid Steam path")
	}
	if items[3] != update {
		t.Error("the update entry is not the one the plugin made")
	}

	s.facts.SteamPathValid = true
	if trayMenu(s, update).Items()[2].Disabled {
		t.Fatal("the Lua folder entry is disabled with a valid Steam path")
	}
}

func TestClickingTheTrayIconTogglesTheWindow(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	s.tray = true
	visible := true
	var hides, shows int
	s.visible = func() bool { return visible }
	s.hide = func() {
		hides++
		visible = false
	}
	s.show = func() {
		shows++
		visible = true
	}
	s.frame = func() {}
	s.notify = func(string) {}

	s.toggle()
	if hides != 1 || visible {
		t.Fatalf("hides = %d, visible = %v, want the shown window hidden", hides, visible)
	}

	s.toggle()
	if shows != 1 || !visible {
		t.Fatalf("shows = %d, visible = %v, want the hidden window shown", shows, visible)
	}
}
