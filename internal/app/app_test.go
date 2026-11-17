package app_test

import (
	"path/filepath"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

func TestLangForFollowsTheLocale(t *testing.T) {
	tests := []struct {
		locale string
		want   app.Lang
	}{
		{"zh-Hans", app.Chinese},
		{"zh-CN", app.Chinese},
		{"zh-Hant-TW", app.Chinese},
		{"ZH", app.Chinese},
		{"en-US", app.English},
		{"de", app.English},
		{"", app.English},
	}
	for _, test := range tests {
		if got := app.LangFor(test.locale); got != test.want {
			t.Errorf("LangFor(%q) = %v, want %v", test.locale, got, test.want)
		}
	}
}

func TestDeploymentFollowsTheLayoutFiles(t *testing.T) {
	root := `C:\Steam`
	holding := func(names ...string) rt.Fake {
		files := map[string]bool{}
		for _, name := range names {
			files[filepath.Join(root, name)] = true
		}
		return rt.Fake{Files: files}
	}
	tests := []struct {
		name  string
		entry string
		files rt.Fake
		want  app.Deployment
	}{
		{
			name:  "default entry",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"),
			want:  app.Applied,
		},
		{
			name:  "early entry selected",
			entry: "version.dll",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "version.dll"),
			want:  app.Applied,
		},
		{
			name:  "missing payload",
			files: holding("xinput1_4.dll", "dwmapi.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "missing secondary loader",
			files: holding("OpenSteamTool.dll", "dwmapi.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "missing entry",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll"),
			want:  app.NotApplied,
		},
		{
			name:  "only the entry that is not selected",
			files: holding("OpenSteamTool.dll", "xinput1_4.dll", "version.dll"),
			want:  app.NotApplied,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := app.DeploymentOf(app.OpenSteamToolLayout(), test.entry, root, test.files); got != test.want {
				t.Errorf("DeploymentOf(%+v) = %v, want %v", test.files.Files, got, test.want)
			}
		})
	}
}

func TestDeploymentNeedsASteamRoot(t *testing.T) {
	if got := app.DeploymentOf(app.OpenSteamToolLayout(), "", "", everything{}); got != app.NotApplied {
		t.Errorf("DeploymentOf against no Steam root = %v, want %v", got, app.NotApplied)
	}
}

type everything struct{}

func (everything) IsFile(string) bool { return true }
