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
