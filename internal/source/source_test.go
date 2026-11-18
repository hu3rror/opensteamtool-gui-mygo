package source_test

import (
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/source"
)

func TestTheThreeBuiltinsLockOneLayoutEach(t *testing.T) {
	tests := []struct {
		key        string
		repository string
		layout     source.Layout
	}{
		{source.KeyOfficial, "OpenSteam001/OpenSteamTool", source.LayoutOpenSteamTool},
		{source.KeyCommunity, "huanyuejue/OpenSteamTool", source.LayoutOpenSteamTool},
		{source.KeySteamDaddy, "Contrary-v2/SteamDaddy-Dll", source.LayoutSteamDaddy},
	}
	builtins := source.Builtins()
	if len(builtins) != len(tests) {
		t.Fatalf("%d built-in sources, want %d", len(builtins), len(tests))
	}
	for i, test := range tests {
		got := builtins[i]
		if got.Key != test.key || got.Repository != test.repository || got.Layout != test.layout || got.Custom {
			t.Errorf("built-in %d is %+v, want %s at %s on layout %v", i, got, test.key, test.repository, test.layout)
		}
	}
}

func TestTheCommunitySourceIsTheDefault(t *testing.T) {
	if got := source.Community(); got.Key != source.KeyCommunity || got.Layout != source.LayoutOpenSteamTool {
		t.Errorf("the default source is %+v, want the community one", got)
	}
	if got := source.OrDefault(source.Source{}); got.Key != source.KeyCommunity {
		t.Errorf("an empty source reads as %+v, want the community one", got)
	}
	custom := source.Source{Key: "https://github.com/someone/OpenSteamTool"}
	if got := source.OrDefault(custom); got != custom {
		t.Errorf("a chosen source reads as %+v, want it kept", got)
	}
}

func TestParseTakesBuiltinKeysAndAddresses(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		key    string
		layout source.Layout
		custom bool
	}{
		{name: "official key", value: "official", key: source.KeyOfficial},
		{name: "community key", value: "community", key: source.KeyCommunity},
		{name: "steamdaddy key", value: "steamdaddy", key: source.KeySteamDaddy, layout: source.LayoutSteamDaddy},
		{name: "key in another case", value: "SteamDaddy", key: source.KeySteamDaddy, layout: source.LayoutSteamDaddy},
		{name: "key with padding", value: " community ", key: source.KeyCommunity},
		{name: "plain address", value: "https://github.com/someone/patch", key: "https://github.com/someone/patch", custom: true},
		{name: "address with the releases suffix", value: "https://github.com/someone/patch/releases", key: "https://github.com/someone/patch", custom: true},
		{name: "address with the latest release suffix", value: "https://github.com/someone/patch/releases/latest", key: "https://github.com/someone/patch", custom: true},
		{name: "address with a release tag suffix", value: "https://github.com/someone/patch/releases/tag/v1.2.3", key: "https://github.com/someone/patch", custom: true},
		{name: "address with a trailing slash", value: "https://github.com/someone/patch/", key: "https://github.com/someone/patch", custom: true},
		{name: "address in another case", value: "HTTPS://GitHub.com/Someone/Patch", key: "https://github.com/Someone/Patch", custom: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := source.Parse(test.value)
			if err != nil {
				t.Fatalf("Parse(%q): %v", test.value, err)
			}
			if got.Key != test.key || got.Layout != test.layout || got.Custom != test.custom {
				t.Errorf("Parse(%q) = %+v, want key %q on layout %v (custom %v)", test.value, got, test.key, test.layout, test.custom)
			}
		})
	}
}

func TestParseRejectsWhatIsNotAGithubRepository(t *testing.T) {
	for _, value := range []string{
		"",
		"   ",
		"gone",
		"github.com/someone/patch",
		"http://github.com/someone/patch",
		"https://gitlab.com/someone/patch",
		"https://github.com/someone",
		"https://github.com/",
		"https://github.com/someone/patch/tree/main",
		"https://github.com/someone/patch/releases/tag/",
		"https://github.com/someone/patch?tab=readme",
		"https://github.com/someone/patch#readme",
		"https://someone" + ":" + "letmein" + "@github.com/someone/patch",
	} {
		if got, err := source.Parse(value); err == nil {
			t.Errorf("Parse(%q) = %+v without a reason", value, got)
		}
	}
}

func TestACustomAddressMatchingABuiltinCollapsesToIt(t *testing.T) {
	tests := []struct {
		address string
		key     string
	}{
		{"https://github.com/OpenSteam001/OpenSteamTool", source.KeyOfficial},
		{"https://github.com/huanyuejue/OpenSteamTool/releases/latest", source.KeyCommunity},
		{"https://github.com/huanyuejue/opensteamtool", source.KeyCommunity},
		{"https://github.com/Contrary-v2/SteamDaddy-Dll/releases", source.KeySteamDaddy},
	}
	for _, test := range tests {
		got, err := source.Parse(test.address)
		if err != nil {
			t.Fatalf("Parse(%q): %v", test.address, err)
		}
		if got.Key != test.key || got.Custom {
			t.Errorf("Parse(%q) = %+v, want the built-in %q", test.address, got, test.key)
		}
	}
}

func TestPatchVersionsAreNormalisedForEquality(t *testing.T) {
	tests := []struct {
		one, other string
		same       bool
	}{
		{"1.4.8", "1.4.8", true},
		{"v1.4.7", "1.4.7", true},
		{"V1.4.7", "1.4.7", true},
		{"  1.4.8  ", "1.4.8", true},
		{"1.4.6-fix", "1.4.6-FIX", true},
		{"1.5.5.4", "v1.5.5.4", true},
		{"v4.0.7", "4.0.7", true},
		{"1.4.8", "v1.4.7", false},
		{"1.4.6-fix", "1.4.6", false},
		{"", "v", true},
	}
	for _, test := range tests {
		if got := source.SameVersion(test.one, test.other); got != test.same {
			t.Errorf("SameVersion(%q, %q) = %v, want %v", test.one, test.other, got, test.same)
		}
	}
	if got := source.NormalizeVersion("  V1.4.6-Fix "); got != "1.4.6-fix" {
		t.Errorf("NormalizeVersion = %q, want %q", got, "1.4.6-fix")
	}
}

func TestTheLayoutsCarryTheirMembers(t *testing.T) {
	tests := []struct {
		layout   source.Layout
		payload  string
		loader   string
		entries  map[source.Entry]string
		optional []string
		forbid   []string
		lua      string
		discard  []string
	}{
		{
			layout:   source.LayoutOpenSteamTool,
			payload:  "OpenSteamTool.dll",
			loader:   "xinput1_4.dll",
			entries:  map[source.Entry]string{source.EntryDefault: "dwmapi.dll", source.EntryEarly: "version.dll"},
			optional: nil,
			forbid:   nil,
			lua:      "config/lua",
			discard:  []string{"*.lib", "*.exp", "lua_static.lib"},
		},
		{
			layout:   source.LayoutSteamDaddy,
			payload:  "SteamDaddy.dll",
			loader:   "xinput1_4.dll",
			entries:  map[source.Entry]string{source.EntryDefault: "version.dll"},
			optional: []string{"cloud_redirect.dll", "steamdaddy.toml"},
			forbid:   []string{"dwmapi.dll"},
			lua:      "config/stplug-in",
			discard:  []string{"ost-manager.exe"},
		},
	}
	for _, test := range tests {
		manifest := test.layout.Manifest()
		if manifest.Payload.Name != test.payload || manifest.Loader.Name != test.loader {
			t.Errorf("layout %v carries payload %q and loader %q, want %q and %q",
				test.layout, manifest.Payload.Name, manifest.Loader.Name, test.payload, test.loader)
		}
		if len(manifest.Entries) != len(test.entries) {
			t.Errorf("layout %v carries %d entries, want %d", test.layout, len(manifest.Entries), len(test.entries))
		}
		for entry, name := range test.entries {
			got, ok := test.layout.EntryFile(entry)
			if !ok || got != name {
				t.Errorf("layout %v entry %v deploys as %q (%v), want %q", test.layout, entry, got, ok, name)
			}
			if !test.layout.Has(entry) {
				t.Errorf("layout %v does not report entry %v", test.layout, entry)
			}
		}
		optional := make([]string, len(manifest.Optional))
		for i, member := range manifest.Optional {
			optional[i] = member.Name
		}
		if !slices.Equal(optional, test.optional) {
			t.Errorf("layout %v optional members %q, want %q", test.layout, optional, test.optional)
		}
		if !slices.Equal(manifest.Forbidden, test.forbid) {
			t.Errorf("layout %v forbids %q, want %q", test.layout, manifest.Forbidden, test.forbid)
		}
		if manifest.Lua != test.lua {
			t.Errorf("layout %v keeps Lua in %q, want %q", test.layout, manifest.Lua, test.lua)
		}
		if !slices.Equal(manifest.Discard, test.discard) {
			t.Errorf("layout %v discards %q, want %q", test.layout, manifest.Discard, test.discard)
		}
	}
}

func TestTheEntryIsOfferedWhereTheSourceAndLayoutAllowIt(t *testing.T) {
	official := source.Source{Key: source.KeyOfficial, Repository: "OpenSteam001/OpenSteamTool", Layout: source.LayoutOpenSteamTool}
	community := source.Community()
	steamDaddy := source.Source{Key: source.KeySteamDaddy, Repository: "Contrary-v2/SteamDaddy-Dll", Layout: source.LayoutSteamDaddy}
	custom := source.Source{Key: "https://github.com/someone/patch", Repository: "someone/patch", Layout: source.LayoutOpenSteamTool, Custom: true}
	tests := []struct {
		source source.Source
		entry  source.Entry
		want   source.EntryState
	}{
		{official, source.EntryDefault, source.EntrySelectable},
		{official, source.EntryEarly, source.EntryUnavailable},
		{community, source.EntryDefault, source.EntrySelectable},
		{community, source.EntryEarly, source.EntrySelectable},
		{steamDaddy, source.EntryDefault, source.EntrySelectable},
		{steamDaddy, source.EntryEarly, source.EntryHidden},
		{custom, source.EntryEarly, source.EntrySelectable},
		{source.Source{}, source.EntryEarly, source.EntrySelectable},
	}
	for _, test := range tests {
		if got := test.source.EntryState(test.entry); got != test.want {
			t.Errorf("%s with entry %v reads as %v, want %v", source.OrDefault(test.source).Key, test.entry, got, test.want)
		}
	}
}

func TestAnEntryTheSourceCannotTakeFallsBackToTheDefault(t *testing.T) {
	official := source.Source{Key: source.KeyOfficial, Repository: "OpenSteam001/OpenSteamTool", Layout: source.LayoutOpenSteamTool}
	steamDaddy := source.Source{Key: source.KeySteamDaddy, Repository: "Contrary-v2/SteamDaddy-Dll", Layout: source.LayoutSteamDaddy}
	tests := []struct {
		source source.Source
		entry  source.Entry
		want   source.Entry
	}{
		{official, source.EntryEarly, source.EntryDefault},
		{steamDaddy, source.EntryEarly, source.EntryDefault},
		{steamDaddy, source.EntryDefault, source.EntryDefault},
		{source.Community(), source.EntryEarly, source.EntryEarly},
	}
	for _, test := range tests {
		if got := source.Effective(test.source, test.entry); got != test.want {
			t.Errorf("%s with entry %v keeps %v, want %v", test.source.Key, test.entry, got, test.want)
		}
	}
}

func TestADraftIsTheSavedSourceUntilItChanges(t *testing.T) {
	community := source.Community()
	custom := source.Source{Key: "https://github.com/someone/patch", Repository: "someone/patch", Layout: source.LayoutOpenSteamTool, Custom: true}

	tests := []struct {
		name  string
		draft source.Draft
		saved source.Source
		same  bool
	}{
		{"the saved built-in", source.Draft{Builtin: source.KeyCommunity}, community, true},
		{"another built-in", source.Draft{Builtin: source.KeySteamDaddy}, community, false},
		{"the saved custom address", source.Draft{Custom: custom.Key}, custom, true},
		{"an empty draft", source.Draft{}, community, false},
		{"an invalid draft", source.Draft{Custom: "github.com/someone/patch"}, community, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.draft.Same(test.saved); got != test.same {
				t.Errorf("the draft reads as the saved source = %v, want %v", got, test.same)
			}
		})
	}
}

func TestADraftCollapsesToTheBuiltinItNames(t *testing.T) {
	draft := source.Draft{Custom: "https://github.com/huanyuejue/OpenSteamTool/releases/latest"}
	resolved, err := draft.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.Key != source.KeyCommunity || resolved.Custom {
		t.Errorf("the draft resolved to %+v, want the community source", resolved)
	}
	if !draft.Same(source.Community()) {
		t.Error("the draft differs from the built-in it names")
	}
	if draft.Layout() != source.LayoutOpenSteamTool {
		t.Errorf("a custom draft reports layout %v, want the OpenSteamTool layout", draft.Layout())
	}
	if got := (source.Draft{Builtin: source.KeySteamDaddy}).Layout(); got != source.LayoutSteamDaddy {
		t.Errorf("the SteamDaddy draft reports layout %v", got)
	}
}
