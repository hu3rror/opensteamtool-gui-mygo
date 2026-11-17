package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

func TestWindowShowsTheDerivedScreenAndFollowsTheLanguage(t *testing.T) {
	w := &window{facts: app.Facts{Lang: app.Chinese}}
	tt := ui.NewTester(w.view, 480, 360)
	if !tt.HasText("未应用") {
		t.Fatalf("texts %q for a snapshot without a Steam path", tt.Texts())
	}
	if !tt.HasText("Steam 路径无效") {
		t.Errorf("texts %q lack the invalid-path explanation", tt.Texts())
	}
	w.facts.Lang = app.English
	tt.Frame()
	if !tt.HasText("Not applied") || tt.HasText("未应用") {
		t.Errorf("texts %q after the language switch", tt.Texts())
	}
}

func TestARealSteamDirectoryWithTheLayoutReadsAsApplied(t *testing.T) {
	steam := t.TempDir()
	for _, name := range []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"} {
		writeFile(t, filepath.Join(steam, name))
	}
	tt := ui.NewTester((&window{facts: factsFor(app.Chinese, steam, rt.OS{})}).view, 480, 360)
	if !tt.HasText("已应用") {
		t.Fatalf("texts %q for %q", tt.Texts(), steam)
	}
	if tt.HasText("Steam 路径无效") {
		t.Errorf("texts %q carry an explanation for a deployed directory", tt.Texts())
	}
}

func TestARealSteamDirectoryWithoutTheLayoutReadsAsNotApplied(t *testing.T) {
	steam := t.TempDir()
	writeFile(t, filepath.Join(steam, "OpenSteamTool.dll"))
	writeFile(t, filepath.Join(steam, "xinput1_4.dll"))
	tt := ui.NewTester((&window{facts: factsFor(app.Chinese, steam, rt.OS{})}).view, 480, 360)
	if !tt.HasText("未应用") {
		t.Fatalf("texts %q for %q", tt.Texts(), steam)
	}
	if tt.HasText("Steam 路径无效") {
		t.Errorf("texts %q call the existing directory invalid", tt.Texts())
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
