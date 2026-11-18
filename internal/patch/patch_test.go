package patch_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
)

func zipped(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

const releaseBody = `{
  "tag_name": "v3.2.1",
  "assets": [
    {"name": "OpenSteamTool-Release.zip", "browser_download_url": "https://example.test/release.zip"},
    {"name": "OpenSteamTool-Debug.zip", "browser_download_url": "https://example.test/debug.zip"}
  ]
}`

func TestTheReleaseCarriesTheVersionAndItsAssets(t *testing.T) {
	release, err := patch.Parse([]byte(releaseBody))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if release.Version != "v3.2.1" {
		t.Errorf("version %q, want the tag", release.Version)
	}
	if len(release.Assets) != 2 || release.Assets[0].URL != "https://example.test/release.zip" {
		t.Fatalf("assets %+v, want both with their URLs", release.Assets)
	}
}

func TestAReleaseThatCannotBeReadIsReported(t *testing.T) {
	if _, err := patch.Parse([]byte("<html>rate limited</html>")); !errors.Is(err, patch.ErrUnparsed) {
		t.Errorf("parse of a page gave %v, want ErrUnparsed", err)
	}
	if _, err := patch.Parse([]byte(`{"assets": []}`)); !errors.Is(err, patch.ErrUnparsed) {
		t.Errorf("a release without a tag gave %v, want ErrUnparsed", err)
	}
}

func TestTheAssetToDownloadFollowsTheNamingRules(t *testing.T) {
	tests := []struct {
		name   string
		assets []string
		want   string
		err    error
	}{
		{
			name:   "the release marker wins over a plain zip",
			assets: []string{"OpenSteamTool.zip", "OpenSteamTool-Release.zip"},
			want:   "OpenSteamTool-Release.zip",
		},
		{
			name:   "the marker wins over an archive",
			assets: []string{"OpenSteamTool.7z", "OpenSteamTool-Release.zip"},
			want:   "OpenSteamTool-Release.zip",
		},
		{
			name:   "a plain zip beats a later marker",
			assets: []string{"SomeOther-Release.zip"},
			want:   "SomeOther-Release.zip",
		},
		{
			name:   "the marker is case insensitive",
			assets: []string{"opensteamtool-release.ZIP"},
			want:   "opensteamtool-release.ZIP",
		},
		{
			name:   "the debug zip is skipped",
			assets: []string{"OpenSteamTool-Debug.zip", "OpenSteamTool-v1.zip"},
			want:   "OpenSteamTool-v1.zip",
		},
		{
			name:   "only a debug zip is no asset",
			assets: []string{"OpenSteamTool-Debug.zip"},
			err:    patch.ErrNoAsset,
		},
		{
			name:   "an archive only release is called that",
			assets: []string{"OpenSteamTool.7z", "OpenSteamTool.rar"},
			err:    patch.ErrArchivesOnly,
		},
		{
			name:   "a tarball counts as an archive",
			assets: []string{"OpenSteamTool.tar.gz"},
			err:    patch.ErrArchivesOnly,
		},
		{
			name:   "a release without assets is no asset",
			assets: nil,
			err:    patch.ErrNoAsset,
		},
		{
			name:   "a checksum beside a debug zip is no asset",
			assets: []string{"OpenSteamTool-Debug.zip", "SHA256SUMS"},
			err:    patch.ErrNoAsset,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assets := make([]patch.Asset, len(test.assets))
			for i, name := range test.assets {
				assets[i] = patch.Asset{Name: name, URL: "https://example.test/" + name}
			}
			got, err := patch.Pick(assets)
			if test.err != nil {
				if !errors.Is(err, test.err) {
					t.Fatalf("Pick = %+v %v, want %v", got, err, test.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Pick: %v", err)
			}
			if got.Name != test.want {
				t.Errorf("picked %q, want %q", got.Name, test.want)
			}
		})
	}
}

func names(files []patch.File) []string {
	got := make([]string, len(files))
	for i, file := range files {
		got[i] = file.Name
	}
	return got
}

func TestExtractKeepsOnlyTheLayoutMembers(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll":     "payload",
		"xinput1_4.dll":         "loader",
		"dwmapi.dll":            "entry",
		"OpenSteamTool.lib":     "import library",
		"lua_static.lib":        "static library",
		"config/lua/extra.lua":  "lua",
		"OpenSteamTool.exp":     "exports",
		"nested/deeper/read.me": "unlisted",
	})
	files, err := patch.Extract(archive, source.LayoutOpenSteamTool, source.EntryDefault)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	want := []string{"OpenSteamTool.dll", "xinput1_4.dll", "dwmapi.dll"}
	if len(files) != len(want) {
		t.Fatalf("extracted %q, want %q", names(files), want)
	}
	for i, file := range files {
		if file.Name != want[i] {
			t.Errorf("member %d is %q, want %q", i, file.Name, want[i])
		}
	}
	if string(files[2].Data) != "entry" {
		t.Errorf("the entry carries %q", files[2].Data)
	}
}

func TestExtractRenamesADisabledEntryBackToItsDeployedName(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll":   "payload",
		"xinput1_4.dll":       "loader",
		"version.dll.disable": "entry",
	})
	files, err := patch.Extract(archive, source.LayoutOpenSteamTool, source.EntryEarly)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if got := names(files); got[len(got)-1] != "version.dll" {
		t.Fatalf("extracted %q, want version.dll", got)
	}
	if string(files[len(files)-1].Data) != "entry" {
		t.Errorf("the renamed member carries %q", files[len(files)-1].Data)
	}
}

func TestExtractTakesTheOptionalMembersWhenTheReleaseHasThem(t *testing.T) {
	archive := zipped(t, map[string]string{
		"SteamDaddy.dll":      "payload",
		"xinput1_4.dll":       "loader",
		"version.dll.disable": "entry",
		"cloud_redirect.dll":  "optional",
		"dwmapi.dll":          "forbidden",
	})
	files, err := patch.Extract(archive, source.LayoutSteamDaddy, source.EntryDefault)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	want := []string{"SteamDaddy.dll", "xinput1_4.dll", "version.dll", "cloud_redirect.dll"}
	got := names(files)
	if len(got) != len(want) {
		t.Fatalf("extracted %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member %d is %q, want %q", i, got[i], want[i])
		}
	}
}

func TestExtractRefusesAPackageWithoutARequiredMember(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"dwmapi.dll":        "entry",
	})
	files, err := patch.Extract(archive, source.LayoutOpenSteamTool, source.EntryDefault)
	if !errors.Is(err, patch.ErrMissingMember) {
		t.Fatalf("Extract = %q %v, want ErrMissingMember", names(files), err)
	}
	if files != nil {
		t.Errorf("a refused package produced %q", names(files))
	}
}

func TestExtractRefusesAPackageWithoutTheSelectedEntry(t *testing.T) {
	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"xinput1_4.dll":     "loader",
		"dwmapi.dll":        "default entry only",
	})
	if _, err := patch.Extract(archive, source.LayoutOpenSteamTool, source.EntryEarly); !errors.Is(err, patch.ErrMissingMember) {
		t.Fatalf("Extract gave %v, want ErrMissingMember", err)
	}
}

func TestExtractRefusesBytesThatAreNotAnArchive(t *testing.T) {
	if _, err := patch.Extract([]byte("<html>not found</html>"), source.LayoutOpenSteamTool, source.EntryDefault); !errors.Is(err, patch.ErrBroken) {
		t.Fatalf("Extract gave %v, want ErrBroken", err)
	}
}

func TestTheCacheShowsAnEmptyAndThenACompleteState(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "community")
	state := patch.Inspect(dir, source.LayoutOpenSteamTool, source.EntryDefault)
	if state.Present || state.Complete || state.Version != "" {
		t.Fatalf("state %+v, want nothing downloaded", state)
	}

	archive := zipped(t, map[string]string{
		"OpenSteamTool.dll": "payload",
		"xinput1_4.dll":     "loader",
		"dwmapi.dll":        "entry",
	})
	files, err := patch.Extract(archive, source.LayoutOpenSteamTool, source.EntryDefault)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if err := patch.Install(dir, files, "v3.2.1"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	state = patch.Inspect(dir, source.LayoutOpenSteamTool, source.EntryDefault)
	if !state.Present || !state.Complete || state.Version != "v3.2.1" {
		t.Fatalf("state %+v, want the downloaded version complete", state)
	}
	if _, err := os.Stat(filepath.Join(dir, patch.VersionFile)); err != nil {
		t.Errorf("the version record is missing: %v", err)
	}
}

func TestLoadReadsTheRequiredMembersAndSkipsTheAbsentOptionals(t *testing.T) {
	dir := t.TempDir()
	held := []patch.File{
		{Name: "SteamDaddy.dll", Data: []byte("payload")},
		{Name: "xinput1_4.dll", Data: []byte("loader")},
		{Name: "version.dll", Data: []byte("entry")},
	}
	if err := patch.Install(dir, held, "v1"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	files, err := patch.Load(dir, source.LayoutSteamDaddy, source.EntryDefault)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := names(files); !slices.Equal(got, []string{"SteamDaddy.dll", "xinput1_4.dll", "version.dll"}) {
		t.Errorf("loaded %q, want the required members only", got)
	}

	withOptional := append(slices.Clone(held), patch.File{Name: "cloud_redirect.dll", Data: []byte("optional")})
	if err := patch.Install(dir, withOptional, "v2"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	files, err = patch.Load(dir, source.LayoutSteamDaddy, source.EntryDefault)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := names(files); len(got) != 4 || !slices.Contains(got, "cloud_redirect.dll") {
		t.Errorf("loaded %q, want the optional member the cache holds", got)
	}

	if _, err := patch.Load(t.TempDir(), source.LayoutSteamDaddy, source.EntryDefault); !errors.Is(err, patch.ErrMissingMember) {
		t.Errorf("Load on an empty cache: %v, want a missing member", err)
	}
}

func TestTheCacheKeepsOnlyTheNewestMemberSet(t *testing.T) {
	dir := t.TempDir()
	first := []patch.File{
		{Name: "OpenSteamTool.dll", Data: []byte("payload")},
		{Name: "xinput1_4.dll", Data: []byte("loader")},
		{Name: "dwmapi.dll", Data: []byte("default entry")},
	}
	if err := patch.Install(dir, first, "v1"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	second := []patch.File{
		{Name: "OpenSteamTool.dll", Data: []byte("payload")},
		{Name: "xinput1_4.dll", Data: []byte("loader")},
		{Name: "version.dll", Data: []byte("early entry")},
	}
	if err := patch.Install(dir, second, "v2"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "dwmapi.dll")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the previous entry is still in the cache: %v", err)
	}
	state := patch.Inspect(dir, source.LayoutOpenSteamTool, source.EntryEarly)
	if !state.Complete || state.Version != "v2" {
		t.Errorf("state %+v, want the newest member set complete", state)
	}
}

func TestACacheWithoutTheEntryFileIsNotComplete(t *testing.T) {
	dir := t.TempDir()
	files := []patch.File{{Name: "OpenSteamTool.dll", Data: []byte("payload")}, {Name: "xinput1_4.dll", Data: []byte("loader")}}
	if err := patch.Install(dir, files, "v1"); err != nil {
		t.Fatalf("Install: %v", err)
	}
	state := patch.Inspect(dir, source.LayoutOpenSteamTool, source.EntryDefault)
	if !state.Present {
		t.Fatalf("state %+v, want the record present", state)
	}
	if state.Complete {
		t.Errorf("state %+v, the entry file is missing", state)
	}
}

func TestEachSourceKeepsItsOwnCacheDirectory(t *testing.T) {
	root := t.TempDir()
	community := patch.Dir(root, source.Community())
	if want := filepath.Join(root, "patches", "community"); community != want {
		t.Errorf("community cache %q, want %q", community, want)
	}
	custom, err := source.Parse("https://github.com/someone/patch")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := patch.Dir(root, custom), filepath.Join(root, "patches", "someone-patch"); got != want {
		t.Errorf("custom cache %q, want %q", got, want)
	}
}

func TestTheClientBuildNumberComesFromTheSteamManifest(t *testing.T) {
	steam := t.TempDir()
	if got := patch.ClientBuild(steam); got != "" {
		t.Errorf("a Steam directory without the manifest reads %q, want nothing", got)
	}
	if err := os.MkdirAll(filepath.Join(steam, "package"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState"
{
	"appid"		"228980"
	"Universe"		"1"
	"version"		"1769731672"
}
`
	if err := os.WriteFile(filepath.Join(steam, "package", "steam_client_win32.manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := patch.ClientBuild(steam); got != "1769731672" {
		t.Errorf("build %q, want the manifest version", got)
	}
}
