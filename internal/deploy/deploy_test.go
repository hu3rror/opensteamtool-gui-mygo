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
	host := &rt.Fake{Dirs: map[string]bool{root: true}, Files: held, Contents: map[string][]byte{
		filepath.Join(root, "steamdaddy.toml"): []byte("default"),
	}}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")
	plan.ToolDir = `C:\Tools`
	plan.Source = community()

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

func steamDaddy() source.Source {
	owner, _ := source.BuiltinFor(source.LayoutSteamDaddy)
	return owner
}

func community() source.Source {
	for _, builtin := range source.Builtins() {
		if builtin.Key == source.KeyCommunity {
			return builtin
		}
	}
	panic("the community source is missing")
}

func userPlan(root, toolDir string) deploy.Plan {
	return deploy.Plan{
		Root:    root,
		ToolDir: toolDir,
		Layout:  source.LayoutSteamDaddy,
		Entry:   source.EntryDefault,
		Source:  steamDaddy(),
		Files: []patch.File{
			{Name: "SteamDaddy.dll", Data: []byte("payload")},
			{Name: "xinput1_4.dll", Data: []byte("loader")},
			{Name: "version.dll", Data: []byte("entry")},
			{Name: "steamdaddy.toml", Data: []byte("default")},
		},
	}
}

func userRemoval(root, toolDir string) deploy.Removal {
	return deploy.Removal{Root: root, ToolDir: toolDir, Layout: source.LayoutSteamDaddy, Source: steamDaddy()}
}

func archived(toolDir, name string) string {
	return filepath.Join(toolDir, "userdata", patch.Slug(source.KeySteamDaddy), name)
}

func content(t *testing.T, host *rt.Fake, path string) string {
	t.Helper()
	data, err := host.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestAnUninstallTakesTheListedRootFilesAndNothingElse(t *testing.T) {
	root := `C:\Steam`
	held := map[string]bool{}
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll", "version.dll", "user.lua"} {
		held[filepath.Join(root, name)] = true
	}
	host := &rt.Fake{Dirs: map[string]bool{root: true, filepath.Join(root, "config", "lua"): true}, Files: held}
	removal := deploy.Removal{Root: root, ToolDir: `C:\Tools`, Layout: source.LayoutOpenSteamTool}

	if err := deploy.Uninstall(host, removal); err != nil {
		t.Fatalf("Uninstall: %v", err)
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
	if err := deploy.Uninstall(host, removal); err != nil {
		t.Errorf("a repeated removal failed on the missing files: %v", err)
	}
}

func TestAnUninstallDeletesAnUnchangedUserFileAndArchivesAnEditedOne(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := userPlan(root, toolDir)
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the unchanged config was archived")
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the config survived the removal")
	}

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write again: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall the edited config: %v", err)
	}
	if got := content(t, host, archived(toolDir, "steamdaddy.toml")); got != "edited" {
		t.Errorf("the archive holds %q, want the edited bytes", got)
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the edited config survived the removal")
	}

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("a repeated removal failed: %v", err)
	}
	if got := content(t, host, archived(toolDir, "steamdaddy.toml")); got != "edited" {
		t.Errorf("the repeated removal left %q, want the archive untouched", got)
	}
}

func TestAFailedArchiveWriteAbortsTheRemoval(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := deploy.Write(host, userPlan(root, toolDir)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")

	if err := deploy.Uninstall(broken{Fake: host, write: "steamdaddy.toml"}, userRemoval(root, toolDir)); err == nil {
		t.Fatal("a failed archive write came back without an error")
	}
	for _, name := range []string{"SteamDaddy.dll", "xinput1_4.dll", "version.dll", "steamdaddy.toml"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s was removed after the failed save", name)
		}
	}
	if len(host.Removes) != 0 {
		t.Errorf("the failed save removed %v", host.Removes)
	}
}

func TestAnUnrecordedUserFileIsArchived(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := host.WriteFile(filepath.Join(root, "steamdaddy.toml"), []byte("mine")); err != nil {
		t.Fatal(err)
	}

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if got := content(t, host, archived(toolDir, "steamdaddy.toml")); got != "mine" {
		t.Errorf("the archive holds %q, want the deployed bytes", got)
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the unrecorded config survived the removal")
	}
}

func TestADeployRestoresTheArchivedConfig(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := userPlan(root, toolDir)
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write again: %v", err)
	}
	if got := content(t, host, filepath.Join(root, "steamdaddy.toml")); got != "edited" {
		t.Errorf("the deploy wrote %q, want the archived bytes", got)
	}
	if !host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the deploy took the archive away")
	}

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall the restored config: %v", err)
	}
	if got := content(t, host, archived(toolDir, "steamdaddy.toml")); got != "edited" {
		t.Errorf("the archive holds %q, the restore changed the shipped baseline", got)
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the restored config survived the removal")
	}
}

func TestADeployWithoutTheMemberRestoresNothing(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := userPlan(root, toolDir)
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	plan.Files = payload("SteamDaddy.dll", "xinput1_4.dll", "version.dll")
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write without the member: %v", err)
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("a member the package no longer carries was restored")
	}
	if !host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the archive was taken away")
	}
}

func TestAnExistingUserFileWinsOverTheArchive(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := userPlan(root, toolDir)
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if err := host.WriteFile(filepath.Join(root, "steamdaddy.toml"), []byte("mine")); err != nil {
		t.Fatal(err)
	}

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write again: %v", err)
	}
	if got := content(t, host, filepath.Join(root, "steamdaddy.toml")); got != "mine" {
		t.Errorf("the deploy wrote %q, want the existing file left alone", got)
	}
}

func TestASwitchArchivesTheOtherLayoutsEditedConfig(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := deploy.Write(host, userPlan(root, toolDir)); err != nil {
		t.Fatalf("Write the SteamDaddy layout: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")
	plan.ToolDir = toolDir
	plan.Source = community()
	plan.Remove = "version.dll"

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("switch the layout: %v", err)
	}
	if got := content(t, host, archived(toolDir, "steamdaddy.toml")); got != "edited" {
		t.Errorf("the archive holds %q, want the other layout's edited config", got)
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) || host.IsFile(filepath.Join(root, "SteamDaddy.dll")) {
		t.Error("the other layout's files survived the switch")
	}
}

func TestASwitchDeletesAnUnchangedUserFile(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := deploy.Write(host, userPlan(root, toolDir)); err != nil {
		t.Fatalf("Write the SteamDaddy layout: %v", err)
	}
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")
	plan.ToolDir = toolDir
	plan.Source = community()
	plan.Remove = "version.dll"

	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("switch the layout: %v", err)
	}
	if host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the unchanged config was archived by the switch")
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the other layout's config survived the switch")
	}
}

func TestAFailedSaveBlocksTheSwitchBeforeAnyRemoval(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := deploy.Write(host, userPlan(root, toolDir)); err != nil {
		t.Fatalf("Write the SteamDaddy layout: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	plan := planFor(root, source.LayoutOpenSteamTool, source.EntryDefault, "OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll")
	plan.ToolDir = toolDir
	plan.Source = community()
	plan.Remove = "version.dll"

	if err := deploy.Write(broken{Fake: host, write: "steamdaddy.toml"}, plan); err == nil {
		t.Fatal("a failed save came back without an error")
	}
	for _, name := range []string{"steamdaddy.toml", "SteamDaddy.dll"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s was removed after the failed save", name)
		}
	}
	if host.IsFile(filepath.Join(root, "OpenSteamTool.dll")) {
		t.Error("the switch went on after the failed save")
	}
}

func TestAnUnreadableUserFileAbortsTheRemoval(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	if err := deploy.Write(host, userPlan(root, toolDir)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	delete(host.Contents, filepath.Join(root, "steamdaddy.toml"))

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err == nil {
		t.Fatal("an unreadable config came back without an error")
	}
	for _, name := range []string{"SteamDaddy.dll", "xinput1_4.dll", "version.dll", "steamdaddy.toml"} {
		if !host.IsFile(filepath.Join(root, name)) {
			t.Errorf("%s was removed after the failed read", name)
		}
	}
	if host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the unreadable config was archived")
	}
}

func TestAnUnchangedUserFileTakesItsStaleArchiveWithIt(t *testing.T) {
	root := `C:\Steam`
	toolDir := `C:\Tools`
	host := &rt.Fake{Dirs: map[string]bool{root: true}}
	plan := userPlan(root, toolDir)
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("edited")
	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Fatal("the edited config was not archived")
	}
	if err := deploy.Write(host, plan); err != nil {
		t.Fatalf("Write again: %v", err)
	}
	host.Contents[filepath.Join(root, "steamdaddy.toml")] = []byte("default")

	if err := deploy.Uninstall(host, userRemoval(root, toolDir)); err != nil {
		t.Fatalf("Uninstall the reset config: %v", err)
	}
	if host.IsFile(archived(toolDir, "steamdaddy.toml")) {
		t.Error("the stale archive survived an unchanged removal")
	}
	if host.IsFile(filepath.Join(root, "steamdaddy.toml")) {
		t.Error("the reset config survived the removal")
	}
}
