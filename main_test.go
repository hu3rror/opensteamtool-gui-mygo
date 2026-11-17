package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

func shown(t *testing.T, s *session) []string {
	t.Helper()
	return ui.NewTester(s.view, 480, 360).Texts()
}

func noSleep(time.Duration) {}

func TestWindowShowsTheDerivedScreenAndFollowsTheLanguage(t *testing.T) {
	s := &session{facts: app.Facts{Lang: app.Chinese}}

	chinese := ui.NewTester(s.view, 480, 360)
	if !slices.Contains(chinese.Texts(), "未应用") {
		t.Fatalf("texts = %q, want 未应用", chinese.Texts())
	}
	if !slices.Contains(chinese.Texts(), "Steam 路径无效") {
		t.Fatalf("texts = %q, want the invalid path explanation", chinese.Texts())
	}

	s.facts.Lang = app.English
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

	s := &session{facts: factsFor(app.Chinese, steam, rt.OS{})}
	if !slices.Contains(shown(t, s), "已应用") {
		t.Fatalf("texts = %q, want 已应用", shown(t, s))
	}
}

func TestARealSteamDirectoryWithoutTheLayoutReadsAsNotApplied(t *testing.T) {
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "steam.exe"))

	s := &session{facts: factsFor(app.Chinese, steam, rt.OS{})}
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
	s := newSession(app.Chinese, root, host, noSleep)
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
	s := newSession(app.Chinese, root, host, noSleep)
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
	s := newSession(app.Chinese, root, host, noSleep)
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
	s := newSession(app.Chinese, root, host, noSleep)
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
	s := newSession(app.Chinese, root, host, noSleep)

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
