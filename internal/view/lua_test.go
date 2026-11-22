package view_test

import (
	"testing"
)

func TestTheLuaTabNamesTheLayoutFolderAndItsState(t *testing.T) {
	s, tt := dialog(t)
	s.facts.LuaRelative = "config/lua"
	s.facts.LuaDir = `D:\Steam\config\lua`
	s.facts.LuaManifest = true
	opened := 0
	s.actions.OpenLua = func() { opened++ }

	if err := tt.Click("Lua 目录"); err != nil {
		t.Fatalf("open the Lua tab: %v", err)
	}
	tt.SetSize(480, 900)
	tt.Frame()
	for _, want := range []string{"布局目录", "config/lua", `D:\Steam\config\lua`, "manifest.lua 已就位"} {
		if !tt.HasText(want) {
			t.Errorf("the Lua tab misses %q", want)
		}
	}
	clickButton(t, tt, "打开 Lua 目录")
	if opened != 1 {
		t.Errorf("the folder was opened %d times, want once", opened)
	}
}

func TestTheLuaTabSaysWhenTheManifestScriptIsAbsent(t *testing.T) {
	s, tt := dialog(t)
	s.facts.LuaRelative = "config/stplug-in"
	s.facts.LuaDir = `D:\Steam\config\stplug-in`

	if err := tt.Click("Lua 目录"); err != nil {
		t.Fatalf("open the Lua tab: %v", err)
	}
	if !tt.HasText("尚无 manifest.lua（可选）") {
		t.Error("the Lua tab hides that manifest.lua is missing")
	}
}

func TestTheLuaTabAsksForASteamPathFirst(t *testing.T) {
	_, tt := dialog(t)
	if err := tt.Click("Lua 目录"); err != nil {
		t.Fatalf("open the Lua tab: %v", err)
	}
	if !tt.HasText("请先指定有效的 Steam 安装路径") {
		t.Error("the Lua tab says nothing without a Steam path")
	}
	if tt.HasText("布局目录") {
		t.Error("the Lua tab names a layout folder without a Steam path")
	}
}
