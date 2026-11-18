package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
)

func host(dirs ...string) *rt.Fake {
	fake := &rt.Fake{Dirs: map[string]bool{}}
	for _, dir := range dirs {
		fake.Dirs[dir] = true
	}
	return fake
}

func write(t *testing.T, dir string, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAMissingConfigIsUnreadable(t *testing.T) {
	_, readable, err := config.Load(t.TempDir(), host())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if readable {
		t.Error("a directory without a config reads as readable")
	}
}

func TestSaveThenLoadCarriesTheChoices(t *testing.T) {
	dir := t.TempDir()
	want := config.Config{SteamPath: `C:\Program Files\Steam`, Language: app.LanguageEnglish}
	if err := config.Save(dir, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, readable, err := config.Load(dir, host(want.SteamPath))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !readable {
		t.Fatal("a saved config reads as unreadable")
	}
	if got != want {
		t.Errorf("loaded %+v, want %+v", got, want)
	}
}

func TestAnEmptySteamPathIsAChoiceAndNotAMissingField(t *testing.T) {
	dir := t.TempDir()
	if err := config.Save(dir, config.Config{Language: app.LanguageAuto}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, readable, err := config.Load(dir, host())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !readable {
		t.Fatal("an empty Steam path made the config unreadable")
	}
	if got.SteamPath != "" || got.Language != app.LanguageAuto {
		t.Errorf("loaded %+v, want the empty path with auto", got)
	}
}

func TestSaveLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	if err := config.Save(dir, config.Config{Language: app.LanguageChinese}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != config.FileName {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("the tool directory holds %q, want only %q", names, config.FileName)
	}
}

func TestAnUnreadableConfigIsNeverPartiallyTaken(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"not json", "{ this is not json"},
		{"empty file", ""},
		{"unsupported version", `{"version": 2, "steam_path": "", "language": "zh"}`},
		{"no version", `{"steam_path": "", "language": "zh"}`},
		{"no language", `{"version": 1, "steam_path": ""}`},
		{"invalid language", `{"version": 1, "steam_path": "", "language": "de"}`},
		{"no steam path", `{"version": 1, "language": "zh"}`},
		{"steam path of the wrong type", `{"version": 1, "steam_path": 3, "language": "zh"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, test.body)
			_, readable, err := config.Load(dir, host())
			if readable {
				t.Error("the config read as readable")
			}
			if err == nil {
				t.Error("an unreadable config came back without a reason")
			}
		})
	}
}

func TestAFieldThisVersionDoesNotUseIsTolerated(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, `{"version": 1, "steam_path": "C:\\Steam", "language": "auto", "theme": "neon", "entry": "early", "patch_source": "gone"}`)
	got, readable, err := config.Load(dir, host(`C:\Steam`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !readable {
		t.Fatal("a field this version does not use made the config unreadable")
	}
	if got.SteamPath != `C:\Steam` || got.Language != app.LanguageAuto {
		t.Errorf("loaded %+v, want the strict fields", got)
	}
}

func TestASavedPathThatIsNotAReadableDirectoryIsUnreadable(t *testing.T) {
	dir := t.TempDir()
	saved := filepath.Join(t.TempDir(), "gone")
	write(t, dir, fmt.Sprintf(`{"version": 1, "steam_path": %s, "language": "auto"}`, strconv.Quote(saved)))

	got, readable, err := config.Load(dir, host())
	if readable {
		t.Fatal("a saved path that is not a directory read as readable")
	}
	if err == nil {
		t.Fatal("a saved path that is not a directory came back without a reason")
	}
	if got.SteamPath != saved {
		t.Fatalf("path = %q, want the saved one kept for the setup screen", got.SteamPath)
	}

	unreadable := t.TempDir()
	write(t, dir, fmt.Sprintf(`{"version": 1, "steam_path": %s, "language": "auto"}`, strconv.Quote(unreadable)))
	locked := &rt.Fake{Dirs: map[string]bool{unreadable: true}, Unreadable: map[string]bool{unreadable: true}}
	if _, readable, _ := config.Load(dir, locked); readable {
		t.Fatal("a saved path that cannot be read read as readable")
	}
}
