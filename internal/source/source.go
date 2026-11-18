package source

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

type Layout int

const (
	LayoutOpenSteamTool Layout = iota
	LayoutSteamDaddy
)

type Entry int

const (
	EntryDefault Entry = iota
	EntryEarly
)

const (
	KeyOfficial   = "official"
	KeyCommunity  = "community"
	KeySteamDaddy = "steamdaddy"
)

type Source struct {
	Key        string
	Repository string
	Layout     Layout
	Custom     bool
}

var builtins = []Source{
	{Key: KeyOfficial, Repository: "OpenSteam001/OpenSteamTool", Layout: LayoutOpenSteamTool},
	{Key: KeyCommunity, Repository: "huanyuejue/OpenSteamTool", Layout: LayoutOpenSteamTool},
	{Key: KeySteamDaddy, Repository: "Contrary-v2/SteamDaddy-Dll", Layout: LayoutSteamDaddy},
}

func Builtins() []Source {
	return slices.Clone(builtins)
}

func Community() Source {
	return builtins[1]
}

func OrDefault(s Source) Source {
	if s.Key == "" {
		return Community()
	}
	return s
}

var ErrAddress = errors.New("a github.com repository address is required")

func Parse(value string) (Source, error) {
	value = strings.TrimSpace(value)
	for _, builtin := range builtins {
		if strings.EqualFold(value, builtin.Key) {
			return builtin, nil
		}
	}
	address, err := NormalizeAddress(value)
	if err != nil {
		return Source{}, err
	}
	return collapse(address), nil
}

func NormalizeAddress(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", ErrAddress
	}
	if !strings.EqualFold(parsed.Scheme, "https") || !strings.EqualFold(parsed.Host, "github.com") {
		return "", ErrAddress
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", ErrAddress
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 || segments[0] == "" || segments[1] == "" {
		return "", ErrAddress
	}
	if len(segments) > 2 && !releaseSuffix(segments[2:]) {
		return "", ErrAddress
	}
	return "https://github.com/" + segments[0] + "/" + segments[1], nil
}

func releaseSuffix(rest []string) bool {
	switch {
	case len(rest) == 1 && rest[0] == "releases":
		return true
	case len(rest) == 2 && rest[0] == "releases" && rest[1] == "latest":
		return true
	case len(rest) == 3 && rest[0] == "releases" && rest[1] == "tag" && rest[2] != "":
		return true
	}
	return false
}

func collapse(address string) Source {
	repository := strings.TrimPrefix(address, "https://github.com/")
	for _, builtin := range builtins {
		if strings.EqualFold(builtin.Repository, repository) {
			return builtin
		}
	}
	return Source{Key: address, Repository: repository, Layout: LayoutOpenSteamTool, Custom: true}
}

func NormalizeVersion(version string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(version)), "v")
}

func SameVersion(one, other string) bool {
	return NormalizeVersion(one) == NormalizeVersion(other)
}

type Member struct {
	Name      string
	Published []string
}

type EntryMember struct {
	Entry
	Member
}

type Manifest struct {
	Payload   Member
	Loader    Member
	Entries   []EntryMember
	Optional  []Member
	Forbidden []string
	Lua       string
	Discard   []string
}

var openSteamToolManifest = Manifest{
	Payload: Member{Name: "OpenSteamTool.dll", Published: []string{"OpenSteamTool.dll"}},
	Loader:  Member{Name: "xinput1_4.dll", Published: []string{"xinput1_4.dll"}},
	Entries: []EntryMember{
		{Entry: EntryDefault, Member: Member{Name: "dwmapi.dll", Published: []string{"dwmapi.dll"}}},
		{Entry: EntryEarly, Member: Member{Name: "version.dll", Published: []string{"version.dll.disable", "version.dll.disabled"}}},
	},
	Lua:     "config/lua",
	Discard: []string{"*.lib", "*.exp", "lua_static.lib"},
}

var steamDaddyManifest = Manifest{
	Payload: Member{Name: "SteamDaddy.dll", Published: []string{"SteamDaddy.dll"}},
	Loader:  Member{Name: "xinput1_4.dll", Published: []string{"xinput1_4.dll"}},
	Entries: []EntryMember{
		{Entry: EntryDefault, Member: Member{Name: "version.dll", Published: []string{"version.dll.disable", "version.dll.disabled"}}},
	},
	Optional: []Member{
		{Name: "cloud_redirect.dll", Published: []string{"cloud_redirect.dll"}},
		{Name: "steamdaddy.toml", Published: []string{"steamdaddy.toml"}},
	},
	Forbidden: []string{"dwmapi.dll"},
	Lua:       "config/stplug-in",
	Discard:   []string{"ost-manager.exe"},
}

func (l Layout) Manifest() Manifest {
	if l == LayoutSteamDaddy {
		return steamDaddyManifest
	}
	return openSteamToolManifest
}

func (l Layout) Has(entry Entry) bool {
	_, ok := l.EntryFile(entry)
	return ok
}

func (l Layout) EntryFile(entry Entry) (string, bool) {
	for _, member := range l.Manifest().Entries {
		if member.Entry == entry {
			return member.Name, true
		}
	}
	return "", false
}

type EntryState int

const (
	EntryHidden EntryState = iota
	EntryUnavailable
	EntrySelectable
)

func (s Source) EntryState(entry Entry) EntryState {
	s = OrDefault(s)
	if !s.Layout.Has(entry) {
		return EntryHidden
	}
	if entry != EntryDefault && s.Key == KeyOfficial {
		return EntryUnavailable
	}
	return EntrySelectable
}

func Effective(s Source, entry Entry) Entry {
	if OrDefault(s).EntryState(entry) == EntrySelectable {
		return entry
	}
	return EntryDefault
}

type Draft struct {
	Builtin string
	Custom  string
}

func (d Draft) Resolve() (Source, error) {
	if strings.TrimSpace(d.Builtin) != "" {
		return Parse(d.Builtin)
	}
	return Parse(d.Custom)
}

func (d Draft) Same(s Source) bool {
	resolved, err := d.Resolve()
	if err != nil {
		return false
	}
	return resolved.Key == OrDefault(s).Key
}

func (d Draft) Layout() Layout {
	if strings.TrimSpace(d.Builtin) != "" {
		for _, builtin := range builtins {
			if builtin.Key == d.Builtin {
				return builtin.Layout
			}
		}
	}
	return LayoutOpenSteamTool
}
