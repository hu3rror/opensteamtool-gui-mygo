package deploy_test

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"opensteamtool-gui-mygo/internal/deploy"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
)

func payload(names ...string) []patch.File {
	files := make([]patch.File, 0, len(names))
	for _, name := range names {
		files = append(files, patch.File{Name: name, Data: []byte(name)})
	}
	return files
}

func planFor(root string, layout source.Layout, entry source.Entry, names ...string) deploy.Plan {
	return deploy.Plan{Root: root, Layout: layout, Entry: entry, Files: payload(names...)}
}

func TestADeployWritesEveryFileByRename(t *testing.T) {
	root := `C:\Steam`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		target := filepath.Join(root, name)
		if !host.IsFile(target) {
			t.Errorf("%s did not land in the Steam directory", name)
		}
		if !slices.Contains(host.Renames, rt.Move{From: target + ".tmp", To: target}) {
			t.Errorf("%s did not arrive by a rename: %v", name, host.Renames)
		}
	}
	if len(host.Writes) != 3 {
		t.Errorf("wrote %d files, want one temp file per member", len(host.Writes))
	}
	for name := range host.Files {
		if strings.HasSuffix(name, ".tmp") {
			t.Errorf("the temp file %q was left behind", name)
		}
	}
	if !slices.Contains(host.Mkdirs, filepath.Join(root, "config", "lua")) {
		t.Errorf("made %q, want the Lua directory", host.Mkdirs)
	}
}

func TestAConflictingEntryBlocksTheWholeWrite(t *testing.T) {
	root := `C:\Steam`
	stale := filepath.Join(root, "version.dll")
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{stale: true}}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")

	err := deploy.Write(host, plan)
	var conflict deploy.Conflict
	if !errors.As(err, &conflict) || conflict.Name != "version.dll" {
		t.Fatalf("Write: %v, want a conflict on version.dll", err)
	}
	if len(host.Writes) != 0 || len(host.Renames) != 0 {
		t.Errorf("the blocked deploy wrote %v and renamed %v", host.Writes, host.Renames)
	}
	if !host.IsFile(stale) {
		t.Error("the blocked deploy removed the conflicting file anyway")
	}
	if len(host.Mkdirs) != 0 {
		t.Errorf("a blocked deploy made %q, want nothing", host.Mkdirs)
	}
}

func TestRemoveAndDeployClearsTheConflict(t *testing.T) {
	root := `C:\Steam`
	stale := filepath.Join(root, "version.dll")
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{stale: true}}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")
	plan.Remove = "version.dll"

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if host.IsFile(stale) {
		t.Error("the conflicting file survived the retry")
	}
	if !host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the deploy did not land after the removal")
	}
}

func TestTheSteamDaddyForbiddenEntryBlocksTheWrite(t *testing.T) {
	root := `C:\Steam`
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{filepath.Join(root, "dwmapi.dll"): true}}
	plan := planFor(root, source.LayoutSteamDaddy, source.EntryDefault, "SteamDaddy.dll", "xinput1_4.dll", "version.dll")

	err := deploy.Write(host, plan)
	var conflict deploy.Conflict
	if !errors.As(err, &conflict) || conflict.Name != "dwmapi.dll" {
		t.Fatalf("Write: %v, want a conflict on dwmapi.dll", err)
	}
	if len(host.Writes) != 0 {
		t.Errorf("the blocked deploy wrote %v", host.Writes)
	}
}

func TestTheOtherLayoutsFilesGoAndNothingElse(t *testing.T) {
	root := `C:\Steam`
	held := map[string]bool{}
	for _, name := range []string{"SteamDaddy.dll", "xinput1_4.dll", "cloud_redirect.dll", "steamdaddy.toml", "notes.txt"} {
		held[filepath.Join(root, name)] = true
	}
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: held}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, name := range []string{"SteamDaddy.dll", "cloud_redirect.dll", "steamdaddy.toml"} {
		if host.IsFile(filepath.Join(root, name)) {
			t.Errorf("the other layout's %s survived the switch", name)
		}
	}
	for _, name := range []string{"xinput1_4.dll", "notes.txt"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s was removed, want it kept", name)
		}
	}
}

func TestAnOptionalFileMarkedKeepIsLeftAlone(t *testing.T) {
	root := `C:\Steam`
	settings := filepath.Join(root, "steamdaddy.toml")
	stale := filepath.Join(root, "cloud_redirect.dll")
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{settings: true, stale: true}}
	plan := planFor(root, source.LayoutSteamDaddy, source.EntryDefault, "SteamDaddy.dll", "xinput1_4.dll", "version.dll", "cloud_redirect.dll", "steamdaddy.toml")

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if slices.ContainsFunc(host.Writes, func(w rt.Write) bool { return w.Path == settings+".tmp" }) {
		t.Errorf("the kept settings file was written: %v", host.Writes)
	}
	if !slices.ContainsFunc(host.Writes, func(w rt.Write) bool { return w.Path == stale+".tmp" }) {
		t.Errorf("the stale optional member was not refreshed: %v", host.Writes)
	}
}

type broken struct {
	*rt.Fake
	write  string
	rename string
}

func (b broken) WriteFile(path string, data []byte) error {
	if b.write != "" && filepath.Base(path) == b.write+".tmp" {
		return errors.New("the disk is full")
	}
	return b.Fake.WriteFile(path, data)
}

func (b broken) Rename(from string, to string) error {
	if b.rename != "" && filepath.Base(to) == b.rename {
		return errors.New("the target is open in another process")
	}
	return b.Fake.Rename(from, to)
}

func TestAMidWayFailureLeavesTheEarlierFileIntact(t *testing.T) {
	root := `C:\Steam`
	host := broken{Fake: &rt.Fake{Dirs: map[string]bool{root: true}}, write: "xinput1_4.dll"}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")

	if err := deploy.Write(host, plan); err == nil {
		t.Fatal("a failed write came back without an error")
	}
	if !host.IsFile(filepath.Join(root, "OpenSteamTool.dll")) {
		t.Error("the file written before the failure is gone")
	}
	if host.IsFile(filepath.Join(root, "dwmapi.dll")) {
		t.Error("the deploy went on after the failure")
	}
}

func TestAFailedRenameLeavesTheOldFileAndTakesTheTempAway(t *testing.T) {
	root := `C:\Steam`
	held := filepath.Join(root, "dwmapi.dll")
	host := broken{Fake: &rt.Fake{Dirs: map[string]bool{root: true}, Files: map[string]bool{held: true}}, rename: "dwmapi.dll"}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")

	if err := deploy.Write(host, plan); err == nil {
		t.Fatal("a failed rename came back without an error")
	}
	if !host.IsFile(held) {
		t.Error("the rename replaced the old file before it failed")
	}
	for name := range host.Files {
		if strings.HasSuffix(name, ".tmp") {
			t.Errorf("the temp file %q was left behind", name)
		}
	}
}

func TestAnUninstallTakesTheListedRootFilesAndNothingElse(t *testing.T) {
	root := `C:\Steam`
	held := map[string]bool{}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll", "version.dll", "user.lua"} {
		held[filepath.Join(root, name)] = true
	}
	host := &rt.Fake{Dirs: map[string]bool{root: true, filepath.Join(root, "config", "lua"): true}, Files: held}

	if err := deploy.Uninstall(host, root, source.LayoutOpenSteamTool); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll", "version.dll"} {
		if host.IsFile(filepath.Join(root, name)) {
			t.Errorf("the layout file %s survived the removal", name)
		}
	}
	if !host.IsFile(filepath.Join(root, "user.lua")) {
		t.Error("the removal took the user's .lua file")
	}
	if !host.IsDir(filepath.Join(root, "config", "lua")) {
		t.Error("the removal took the Lua directory")
	}
	if err := deploy.Uninstall(host, root, source.LayoutOpenSteamTool); err != nil {
		t.Errorf("a repeated removal failed on the missing files: %v", err)
	}
}
