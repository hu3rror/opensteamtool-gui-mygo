package rt_test

import (
	"os"
	"path/filepath"
	"testing"

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
