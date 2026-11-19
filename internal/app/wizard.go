package app

type WizardStep int

const (
	WizardLanguage WizardStep = iota
	WizardTheme
	WizardPath
	WizardDownload
)

type WizardDownloadState int

const (
	WizardNotDownloaded WizardDownloadState = iota
	WizardDownloaded
	WizardDownloading
	WizardDownloadFailed
)

type Wizard struct {
	Open     bool
	Step     WizardStep
	Language Language
	Theme    Theme
	Download WizardDownloadState
	Failure  Result
}

type WizardView struct {
	Open         bool
	Step         WizardStep
	Language     Language
	Theme        Theme
	Download     WizardDownloadState
	Failure      Result
	NextDisabled bool
}

func wizardViewOf(w Wizard, pathValid bool) WizardView {
	return WizardView{
		Open:         w.Open,
		Step:         w.Step,
		Language:     w.Language,
		Theme:        w.Theme,
		Download:     w.Download,
		Failure:      w.Failure,
		NextDisabled: w.Step == WizardPath && !pathValid,
	}
}
