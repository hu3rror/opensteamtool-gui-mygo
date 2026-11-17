package app

import (
	"path/filepath"
	"strings"
)

type Lang int

const (
	Chinese Lang = iota
	English
)

func LangFor(locale string) Lang {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		return Chinese
	}
	return English
}

type MainState int

const (
	MainApplied MainState = iota
	MainNotApplied
	MainPatchFailed
	MainApplying
	MainRemoving
	MainStartingSteam
	MainStoppingSteam
)

type Explanation int

const (
	NoExplanation Explanation = iota
	InvalidSteamPath
	PatchNotDownloaded
)

type Busy int

const (
	Idle Busy = iota
	BusyCheckUpdate
	BusyDownloadPatch
	BusyApplyPatch
	BusyUninstall
	BusyStartSteam
	BusyStopSteam
)

var stageStates = map[Busy]MainState{
	BusyApplyPatch: MainApplying,
	BusyUninstall:  MainRemoving,
	BusyStartSteam: MainStartingSteam,
	BusyStopSteam:  MainStoppingSteam,
}

type Deployment int

const (
	NotApplied Deployment = iota
	Applied
)

type Layout struct {
	Payload         string
	SecondaryLoader string
	DefaultEntry    string
}

func OpenSteamToolLayout() Layout {
	return Layout{
		Payload:         "OpenSteamTool.dll",
		SecondaryLoader: "xinput1_4.dll",
		DefaultEntry:    "dwmapi.dll",
	}
}

type Files interface {
	IsFile(path string) bool
}

func DeploymentOf(layout Layout, entry string, root string, files Files) Deployment {
	if root == "" {
		return NotApplied
	}
	if !files.IsFile(filepath.Join(root, layout.Payload)) || !files.IsFile(filepath.Join(root, layout.SecondaryLoader)) {
		return NotApplied
	}
	if entry == "" {
		entry = layout.DefaultEntry
	}
	if files.IsFile(filepath.Join(root, entry)) {
		return Applied
	}
	return NotApplied
}

type Facts struct {
	Lang            Lang
	SteamPathValid  bool
	Deployment      Deployment
	PatchDownloaded bool
	PatchFailed     bool
	Busy            Busy
}

type View struct {
	Lang        Lang
	MainState   MainState
	Explanation Explanation
}

func Derive(f Facts) View {
	v := View{Lang: f.Lang, MainState: mainStateOf(f)}
	if f.Busy == Idle {
		v.Explanation = explanationOf(f)
	}
	return v
}

func mainStateOf(f Facts) MainState {
	if state, ok := stageStates[f.Busy]; ok {
		return state
	}
	if f.PatchFailed {
		return MainPatchFailed
	}
	if f.SteamPathValid && f.Deployment == Applied {
		return MainApplied
	}
	return MainNotApplied
}

func explanationOf(f Facts) Explanation {
	switch {
	case !f.SteamPathValid:
		return InvalidSteamPath
	case !f.PatchDownloaded:
		return PatchNotDownloaded
	}
	return NoExplanation
}
