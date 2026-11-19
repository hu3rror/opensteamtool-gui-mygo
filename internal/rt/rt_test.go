package rt_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
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

func TestOSFetchesABodyOverHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/release.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"v1"}`))
	}))
	defer server.Close()

	osRuntime := rt.OS{}
	body, err := osRuntime.Get(server.URL + "/release.json")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(body) != `{"tag_name":"v1"}` {
		t.Errorf("body %q, want the served document", body)
	}
	if _, err := osRuntime.Get(server.URL + "/missing"); err == nil {
		t.Error("a missing document came back without an error")
	}
}

func TestTheFakeAnswersFromItsResponses(t *testing.T) {
	fake := &rt.Fake{Responses: map[string][]byte{"https://example.test/release": []byte("body")}}
	body, err := fake.Get("https://example.test/release")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(body) != "body" {
		t.Errorf("body %q", body)
	}
	if !slices.Equal(fake.Fetched, []string{"https://example.test/release"}) {
		t.Errorf("fetched %q, want the URL once", fake.Fetched)
	}
	if _, err := fake.Get("https://example.test/other"); err == nil {
		t.Error("a URL the fake was not given came back without an error")
	}

	fake.FetchError = errors.New("offline")
	if _, err := fake.Get("https://example.test/release"); err == nil {
		t.Error("the offline fake answered anyway")
	}
}

func TestOSFetchReportsTheStatusAndTheTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			time.Sleep(50 * time.Millisecond)
		}
		if r.URL.Path != "/signature" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("sig"))
	}))
	defer server.Close()

	osRuntime := rt.OS{}
	response, err := osRuntime.Fetch(server.URL+"/signature", time.Second)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if response.Status != 200 || string(response.Body) != "sig" {
		t.Errorf("response %d %q, want the served signature", response.Status, response.Body)
	}
	missing, err := osRuntime.Fetch(server.URL+"/missing", time.Second)
	if err != nil {
		t.Fatalf("a 404 came back as an error: %v", err)
	}
	if missing.Status != 404 {
		t.Errorf("status %d, want the 404 itself", missing.Status)
	}
	if _, err := osRuntime.Fetch(server.URL+"/slow", 10*time.Millisecond); err == nil {
		t.Error("a request past its timeout answered anyway")
	}
}

func TestTheFakeFetchAnswersWithTheStatusItWasGiven(t *testing.T) {
	fake := &rt.Fake{Answers: map[string]rt.Answer{
		"https://example.test/hit":  {Body: []byte("sig")},
		"https://example.test/none": {Status: 404},
		"https://example.test/down": {Err: errors.New("offline")},
	}}
	hit, err := fake.Fetch("https://example.test/hit", time.Second)
	if err != nil || hit.Status != 200 || string(hit.Body) != "sig" {
		t.Errorf("hit = %d %q %v, want the body with a 200", hit.Status, hit.Body, err)
	}
	none, err := fake.Fetch("https://example.test/none", time.Second)
	if err != nil || none.Status != 404 {
		t.Errorf("none = %d %v, want the 404 without an error", none.Status, err)
	}
	if _, err := fake.Fetch("https://example.test/down", time.Second); err == nil {
		t.Error("the failing answer came back without an error")
	}
	if _, err := fake.Fetch("https://example.test/unknown", time.Second); err == nil {
		t.Error("a URL the fake was not given came back without an error")
	}
	if len(fake.Fetched) != 4 {
		t.Errorf("fetched %q, want every request recorded", fake.Fetched)
	}
}

func TestOSMakesWritesRenamesAndRemovesFiles(t *testing.T) {
	dir := t.TempDir()
	osRuntime := rt.OS{}
	lua := filepath.Join(dir, "config", "lua")
	if err := osRuntime.MkdirAll(lua); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if !osRuntime.IsDir(lua) {
		t.Errorf("%q did not become a directory", lua)
	}
	temp := filepath.Join(dir, "dwmapi.dll.tmp")
	if err := osRuntime.WriteFile(temp, []byte("entry")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	file := filepath.Join(dir, "dwmapi.dll")
	if err := osRuntime.Rename(temp, file); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if osRuntime.IsFile(temp) || !osRuntime.IsFile(file) {
		t.Errorf("the rename left %q and %q", temp, file)
	}
	if data, err := osRuntime.ReadFile(file); err != nil || string(data) != "entry" {
		t.Errorf("ReadFile: %q, %v", data, err)
	}
	if err := osRuntime.Remove(file); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if osRuntime.IsFile(file) {
		t.Errorf("%q survived the removal", file)
	}
	if _, err := osRuntime.ReadFile(file); err == nil {
		t.Error("a removed file read back without an error")
	}
}

func TestTheFakeRecordsItsFileChanges(t *testing.T) {
	fake := &rt.Fake{}
	file := `C:\Steam\dwmapi.dll`
	if err := fake.WriteFile(file+".tmp", []byte("entry")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := fake.Rename(file+".tmp", file); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if !fake.IsFile(file) || fake.IsFile(file+".tmp") {
		t.Errorf("the rename left %v", fake.Files)
	}
	if data, err := fake.ReadFile(file); err != nil || string(data) != "entry" {
		t.Errorf("ReadFile: %q, %v", data, err)
	}
	if _, err := fake.ReadFile(`C:\Steam\missing.dll`); err == nil {
		t.Error("a missing file read back without an error")
	}
	fake.Files[`C:\Steam\opaque.dll`] = true
	if _, err := fake.ReadFile(`C:\Steam\opaque.dll`); err == nil {
		t.Error("a file without content read back without an error")
	}
	if len(fake.Writes) != 1 || fake.Writes[0].Path != file+".tmp" || string(fake.Writes[0].Data) != "entry" {
		t.Errorf("wrote %+v", fake.Writes)
	}
	if !slices.Equal(fake.Renames, []rt.Move{{From: file + ".tmp", To: file}}) {
		t.Errorf("renamed %+v", fake.Renames)
	}
	if err := fake.Remove(file); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if fake.IsFile(file) || !slices.Equal(fake.Removes, []string{file}) {
		t.Errorf("removes %v with %v left", fake.Removes, fake.Files)
	}
	if err := fake.MkdirAll(`C:\Steam\config\lua`); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if !fake.IsDir(`C:\Steam\config\lua`) {
		t.Errorf("mkdirs %v", fake.Mkdirs)
	}
}

func TestTheFakeReportsEveryOpenedPath(t *testing.T) {
	fake := &rt.Fake{}
	if err := fake.OpenPath(`C:\Steam\config\lua`); err != nil {
		t.Fatalf("OpenPath: %v", err)
	}
	if !slices.Equal(fake.Opens, []string{`C:\Steam\config\lua`}) {
		t.Errorf("opened %v", fake.Opens)
	}
	fake.OpenError = errors.New("no handler")
	if err := fake.OpenPath(`C:\Steam\config\lua`); !errors.Is(err, fake.OpenError) {
		t.Errorf("the failure came back as %v", err)
	}
	if len(fake.Opens) != 2 {
		t.Errorf("recorded %v, want both attempts", fake.Opens)
	}
}
