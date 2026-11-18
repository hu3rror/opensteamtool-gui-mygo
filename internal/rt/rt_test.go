package rt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"opensteamtool-gui-mygo/internal/rt"
)

func TestOSAnswersFromTheDisk(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "steam.exe")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	osRuntime := rt.OS{}
	if !osRuntime.IsDir(dir) {
		t.Errorf("%q is not reported as a directory", dir)
	}
	if osRuntime.IsDir(file) {
		t.Errorf("%q is reported as a directory", file)
	}
	if osRuntime.IsDir(filepath.Join(dir, "missing")) {
		t.Error("a missing path is reported as a directory")
	}
}

func TestFakeAnswersWhatItIsGiven(t *testing.T) {
	fake := rt.Fake{Dirs: map[string]bool{`C:\Steam`: true}}
	if !fake.IsDir(`C:\Steam`) {
		t.Error("the given directory is not reported")
	}
	if fake.IsDir(`C:\Other`) {
		t.Error("a path the fake was not given is reported")
	}
}

func TestOSSeesFilesAndNotDirectories(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "OpenSteamTool.dll")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(dir, "dwmapi.dll")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	osRuntime := rt.OS{}
	if !osRuntime.IsFile(file) {
		t.Errorf("%q is not reported as a file", file)
	}
	if osRuntime.IsFile(folder) {
		t.Errorf("the directory %q is reported as a file", folder)
	}
	if osRuntime.IsFile(filepath.Join(dir, "missing.dll")) {
		t.Error("a missing file is reported as a file")
	}
}

func TestFakeSeesTheFilesItIsGiven(t *testing.T) {
	fake := rt.Fake{Files: map[string]bool{`C:\Steam\OpenSteamTool.dll`: true}}
	if !fake.IsFile(`C:\Steam\OpenSteamTool.dll`) {
		t.Error("the given file is not reported")
	}
	if fake.IsFile(`C:\Steam\dwmapi.dll`) {
		t.Error("a file the fake was not given is reported")
	}
}

func TestOSListsAndStopsAProcessItStarted(t *testing.T) {
	osRuntime := rt.OS{}
	pid, err := osRuntime.Spawn("ping.exe", "", "-n", "60", "127.0.0.1")
	if err != nil {
		t.Fatalf("cannot start ping.exe: %v", err)
	}
	defer func() { _ = osRuntime.Terminate(pid) }()

	exe, listed := processExe(osRuntime, pid)
	if !listed {
		t.Fatalf("process %d is not listed", pid)
	}
	if !strings.EqualFold(filepath.Base(exe), "ping.exe") {
		t.Errorf("the image of process %d is %q, want ping.exe", pid, exe)
	}

	if err := osRuntime.Terminate(pid); err != nil {
		t.Fatalf("cannot terminate process %d: %v", pid, err)
	}
	for waited := time.Duration(0); waited < 5*time.Second; waited += 100 * time.Millisecond {
		if _, listed := processExe(osRuntime, pid); !listed {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("process %d is still listed after termination", pid)
}

func processExe(osRuntime rt.OS, pid int) (string, bool) {
	processes, err := osRuntime.Processes()
	if err != nil {
		return "", false
	}
	for _, process := range processes {
		if process.PID == pid {
			return process.Exe, true
		}
	}
	return "", false
}

func TestOSReadsADirectoryOnlyWhenItCanListIt(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "steam.exe")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	osRuntime := rt.OS{}
	if !osRuntime.IsReadableDir(dir) {
		t.Errorf("%q is not readable", dir)
	}
	if osRuntime.IsReadableDir(file) {
		t.Errorf("the file %q reads as a readable directory", file)
	}
	if osRuntime.IsReadableDir(filepath.Join(dir, "missing")) {
		t.Error("a missing directory reads as readable")
	}
}

func TestTheFakeKeepsADirectoryUnreadableWhenToldTo(t *testing.T) {
	fake := rt.Fake{
		Dirs:       map[string]bool{`C:\Steam`: true, `C:\Locked`: true},
		Unreadable: map[string]bool{`C:\Locked`: true},
	}
	if !fake.IsReadableDir(`C:\Steam`) {
		t.Error("a given directory is not readable")
	}
	if fake.IsReadableDir(`C:\Locked`) {
		t.Error("the locked directory reads as readable")
	}
	if fake.IsReadableDir(`C:\Missing`) {
		t.Error("a path the fake was not given reads as readable")
	}
}

func TestTheFakePicksTheDirectoryItWasGiven(t *testing.T) {
	fake := &rt.Fake{Picked: `C:\Steam`}
	got, err := fake.PickDirectory("Steam directory", `C:\`)
	if err != nil {
		t.Fatalf("PickDirectory: %v", err)
	}
	if got != `C:\Steam` {
		t.Errorf("picked %q, want the given directory", got)
	}
	if fake.PickStart != `C:\` {
		t.Errorf("the dialog started at %q", fake.PickStart)
	}
}
