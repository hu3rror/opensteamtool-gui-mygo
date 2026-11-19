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

type Language int

const (
	LanguageAuto Language = iota
	LanguageChinese
	LanguageEnglish
)

func (l Language) Resolve(locale string) Lang {
	switch l {
	case LanguageChinese:
		return Chinese
	case LanguageEnglish:
		return English
	}
	return LangFor(locale)
}

type Tab int

const (
	TabGeneral Tab = iota
	TabPatchSource
	TabSteam
	TabLua
	TabAbout
)

func Tabs() []Tab {
	return []Tab{TabGeneral, TabPatchSource, TabSteam, TabLua, TabAbout}
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

type Confirmation int

const (
	NoConfirmation Confirmation = iota
	ConfirmExitRemove
	ConfirmRemoveAndRestart
	ConfirmExit
)

type SecondaryAction int

const (
	SecondaryUninstall SecondaryAction = iota
	SecondaryExitAndUninstall
	SecondaryUninstallAndRestart
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

func Stage(busy Busy) (MainState, bool) {
	state, ok := stageStates[busy]
	return state, ok
}

type Gate struct {
	kind Busy
}

func (g *Gate) Busy() Busy {
	return g.kind
}

func (g *Gate) Try(kind Busy) bool {
	if g.kind != Idle || kind == Idle {
		return false
	}
	g.kind = kind
	return true
}

func (g *Gate) Advance(kind Busy) {
	if g.kind != Idle {
		g.kind = kind
	}
}

func (g *Gate) Release() {
	g.kind = Idle
}

type Result int

const (
	NoResult Result = iota
	SteamStarted
	SteamRestarted
	PatchApplied
	PatchRemoved
	RemovalFailed
	PathUnset
	MissingSteamProgram
	CloseSteamFailed
	StartSteamFailed
	DeploymentFailed
	Downloaded
	NetworkFailed
	NoAsset
	ArchivesOnly
	MissingMember
	ParseFailed
	LocalIOFailed
	WriteFailed
)

type Status struct {
	SteamRunning bool
	Busy         Busy
	Result       Result
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

type Conclusion int

const (
	NoConclusion Conclusion = iota
	UpToDate
	NewPatch
	CheckFailed
	ClientUpdated
)

type Facts struct {
	Language        Language
	Locale          string
	NeedsSetup      bool
	SteamPath       string
	SteamPathValid  bool
	SteamRunning    bool
	Deployment      Deployment
	PatchDownloaded bool
	PatchComplete   bool
	PatchVersion    string
	PatchFailed     bool
	Conflict        string
	OnlineVersion   string
	UpdateChecked   bool
	UpdateFailed    bool
	SnapshotBuild   string
	ClientBuild     string
	Busy            Busy
	Result          Result
	Pending         Confirmation
}

type View struct {
	Lang            Lang
	NeedsSetup      bool
	MainState       MainState
	Explanation     Explanation
	Status          Status
	Conflict        string
	Update          Conclusion
	DownloadOffered bool
	PrimaryDisabled bool
	UpdateInFlight  Busy
	Secondary       []SecondaryAction
	Pending         Confirmation
}

func Derive(f Facts) View {
	v := View{Lang: f.Language.Resolve(f.Locale), NeedsSetup: f.NeedsSetup, MainState: mainStateOf(f), Conflict: f.Conflict, Status: Status{SteamRunning: f.SteamRunning, Result: f.Result}, Secondary: secondaryOf(f), Pending: f.Pending}
	v.Update, v.DownloadOffered = conclusionOf(f)
	v.PrimaryDisabled = !f.SteamPathValid || !f.PatchDownloaded || !f.PatchComplete
	if _, ok := Stage(f.Busy); ok {
		v.Status.Busy = f.Busy
	} else {
		v.UpdateInFlight = f.Busy
	}
	if f.Busy == Idle {
		v.Explanation = explanationOf(f)
	}
	return v
}

func conclusionOf(f Facts) (Conclusion, bool) {
	if !f.UpdateChecked && !f.UpdateFailed {
		return NoConclusion, false
	}
	if f.UpdateFailed || f.OnlineVersion == "" {
		return CheckFailed, false
	}
	complete := f.PatchDownloaded && f.PatchComplete
	stale := !complete || !SameVersion(f.PatchVersion, f.OnlineVersion)
	clientMoved := f.SnapshotBuild != "" && f.ClientBuild != "" && f.SnapshotBuild != f.ClientBuild
	if f.PatchDownloaded && clientMoved {
		return ClientUpdated, stale
	}
	if stale {
		return NewPatch, true
	}
	return UpToDate, false
}

func SameVersion(left string, right string) bool {
	return normalizeVersion(left) == normalizeVersion(right)
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v)), "v")
}

func mainStateOf(f Facts) MainState {
	if state, ok := Stage(f.Busy); ok {
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

func secondaryOf(f Facts) []SecondaryAction {
	if !f.SteamPathValid || f.Deployment != Applied {
		return nil
	}
	if f.SteamRunning {
		return []SecondaryAction{SecondaryExitAndUninstall, SecondaryUninstallAndRestart}
	}
	return []SecondaryAction{SecondaryUninstall, SecondaryUninstallAndRestart}
}
