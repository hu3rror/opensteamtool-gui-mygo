package view

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/source"
)

type State struct {
	SettingsOpen bool
	Tab          int
	Language     int
	Theme        int
	PathDraft    string
	PathFocused  bool
	PathError    bool
	PathHandle   ui.Handle

	Source        source.Source
	SourceChoice  string
	SourceCustom  string
	SourceError   bool
	SourceFocused bool
	SourceHandle  ui.Handle

	Entry source.Entry

	CompatDetails bool

	WizardLanguage int
	WizardTheme    int
}

type Actions struct {
	SetLanguage         func(app.Language)
	SetTheme            func(app.Theme)
	CommitPath          func(raw string) bool
	Browse              func()
	ApplySource         func(draft source.Draft) (source.Source, bool)
	SetEntry            func(source.Entry)
	CheckUpdate         func()
	Download            func()
	Apply               func()
	RemoveConflict      func()
	Uninstall           func()
	ExitAndUninstall    func()
	UninstallAndRestart func()
	Confirm             func()
	Cancel              func()
	Prewarm             func()
	CheckToolUpdate     func()
	OpenProject         func()
	OpenDownload        func()
	WizardNext          func()
	WizardBack          func()
	WizardRerun         func()
	WizardLanguage      func(app.Language)
	WizardTheme         func(app.Theme)
	WizardDownload      func()
	WizardSkip          func()
	WizardFinish        func()
}

func Main(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(2)).Children(func() {
		if v.Wizard.Open {
			wizard(c, v, st, a)
			return
		}
		mainScreen(c, v, st, a)
	})
	if st.SettingsOpen {
		settingsDialog(c, v, st, a)
	}
	confirm(c, v, a)
}

func wizard(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Children(func() {
		ui.Text(c, text(v.Lang, wizardTitle)).TextColor(t.TextMuted)
		ui.Spacer(c)
		ui.Text(c, WizardStepLabel(v.Lang, v.Wizard.Step)).TextColor(t.TextMuted)
	})
	ui.Text(c, text(v.Lang, wizardStepNames[v.Wizard.Step])).Role(ui.RoleStatus).FontSize(t.FontSize * 1.5)
	ui.Text(c, text(v.Lang, wizardStepHints[v.Wizard.Step])).TextColor(t.TextMuted)
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			switch v.Wizard.Step {
			case app.WizardLanguage:
				wizardLanguage(c, v, st, a)
			case app.WizardTheme:
				wizardTheme(c, v, st, a)
			case app.WizardPath:
				wizardPath(c, v, st, a)
			case app.WizardDownload:
				downloadShape(c, v)
			}
		})
	})
	wizardFooter(c, v, a)
}

func wizardLanguage(c *ui.Context, v app.View, st *State, a Actions) {
	labels := choiceLabels(v.Lang, followSystem, languageChinese, languageEnglish)
	value := labels[st.WizardLanguage]
	if ui.Select(c, &value, labels).Changed() {
		a.WizardLanguage(app.Language(slices.Index(labels, value)))
	}
	t := c.Theme()
	ui.Column(c).Padding(t.Space(3)).Gap(t.Space(2)).Background(t.Surface).Border(1, t.Border).Radius(t.Radius).Children(func() {
		ui.Text(c, text(v.Lang, applied)).TextColor(t.Success)
		ui.Text(c, text(v.Lang, applyAndStart))
	})
}

func wizardTheme(c *ui.Context, v app.View, st *State, a Actions) {
	labels := choiceLabels(v.Lang, followSystem, themeDark, themeLight)
	value := labels[st.WizardTheme]
	if ui.Select(c, &value, labels).Changed() {
		a.WizardTheme(app.Theme(slices.Index(labels, value)))
	}
	t := c.Theme()
	ui.Row(c).Gap(t.Space(3)).Children(func() {
		themePreview(c, v, ui.DarkTheme(), themeDark, st.WizardTheme == int(app.ThemeDark))
		themePreview(c, v, ui.LightTheme(), themeLight, st.WizardTheme == int(app.ThemeLight))
	})
}

func themePreview(c *ui.Context, v app.View, look *ui.Theme, label key, chosen bool) {
	t := c.Theme()
	border, width := t.Border, float32(1)
	if chosen {
		border, width = t.Accent, 2
	}
	ui.Column(c).Grow(1).Padding(t.Space(3)).Gap(t.Space(2)).Background(look.Background).Border(width, border).Radius(t.Radius).Children(func() {
		ui.Text(c, text(v.Lang, applied)).TextColor(look.Text)
		ui.Text(c, text(v.Lang, label)).TextColor(look.TextMuted)
	})
}

func wizardFooter(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Children(func() {
		wizardBack(c, v, a)
		if v.Wizard.Step == app.WizardDownload && v.Wizard.Download != app.WizardDownloaded {
			if ui.Button(c, text(v.Lang, wizardSkip)).Clicked() {
				a.WizardSkip()
			}
		}
		ui.Spacer(c)
		switch v.Wizard.Step {
		case app.WizardDownload:
			switch v.Wizard.Download {
			case app.WizardNotDownloaded:
				if ui.Button(c, text(v.Lang, wizardDownload)).Clicked() {
					a.WizardDownload()
				}
			case app.WizardDownloaded:
				if ui.Button(c, text(v.Lang, wizardFinish)).Clicked() {
					a.WizardFinish()
				}
			case app.WizardDownloadFailed:
				if ui.Button(c, text(v.Lang, wizardRetry)).Clicked() {
					a.WizardDownload()
				}
			}
		default:
			if ui.Button(c, text(v.Lang, nextStep)).Disabled(v.Wizard.NextDisabled).Clicked() {
				a.WizardNext()
			}
		}
	})
}

func choiceLabels(l app.Lang, keys ...key) []string {
	labels := make([]string, len(keys))
	for i, k := range keys {
		labels[i] = text(l, k)
	}
	return labels
}

func wizardBack(c *ui.Context, v app.View, a Actions) {
	if v.Wizard.Step == app.WizardLanguage {
		return
	}
	if ui.Button(c, text(v.Lang, backStep)).Clicked() {
		a.WizardBack()
	}
}

func wizardPath(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	pathField(c, v, st, a, text(v.Lang, wizardPathPrompt), wizardPathError, "")
	if ui.Button(c, text(v.Lang, browse)).Clicked() {
		a.Browse()
	}
	sourceChoices(c, v, st)
	ui.Text(c, text(v.Lang, fileLayout)).TextColor(t.TextMuted)
	ui.Text(c, text(v.Lang, layoutLabels[st.Draft().Layout()])).TextColor(t.TextMuted)
}

func sourceChoices(c *ui.Context, v app.View, st *State) {
	ui.RadioGroup(c, func() {
		for _, builtin := range source.Builtins() {
			if ui.Radio(c, &st.SourceChoice, builtin.Key, text(v.Lang, sourceLabels[builtin.Key])).Changed() {
				st.SourceError = false
			}
		}
		if ui.Radio(c, &st.SourceChoice, CustomChoice, text(v.Lang, customRepository)).Changed() {
			st.SourceError = false
		}
	})
	if st.SourceChoice != CustomChoice {
		return
	}
	submitted := false
	field := ui.Field(c, text(v.Lang, customSourceHint), func() {
		input := ui.TextInput(c, &st.SourceCustom).Bind(&st.SourceHandle)
		submitted = input.Submitted()
	})
	if st.SourceError {
		field.Error(text(v.Lang, addressError))
	}
	focused := st.SourceHandle.Focused(c)
	if submitted || (st.SourceFocused && !focused) {
		st.commitSource()
	}
	st.SourceFocused = focused
}

func downloadShape(c *ui.Context, v app.View) {
	t := c.Theme()
	w := v.Wizard
	switch w.Download {
	case app.WizardNotDownloaded:
		ui.Text(c, text(v.Lang, wizardNotDownloaded))
	case app.WizardDownloaded:
		ui.Text(c, text(v.Lang, wizardDownloaded))
	case app.WizardDownloading:
		ui.Text(c, text(v.Lang, wizardDownloading))
	case app.WizardDownloadFailed:
		ui.Text(c, WizardDownloadFailure(v.Lang, w.Failure)).Role(ui.RoleStatus).TextColor(t.Warning)
	}
}

func mainScreen(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(t.Space(4)).Children(func() {
		ui.Text(c, text(v.Lang, mainStateKeys[v.MainState])).Role(ui.RoleStatus).FontSize(t.FontSize * 1.5)
		ui.Spacer(c)
		if ui.Button(c, text(v.Lang, settings)).Clicked() {
			st.SettingsOpen = true
		}
	})
	if copy, ok := explanationCopies[v.Explanation]; ok {
		ui.Text(c, text(v.Lang, copy.title))
		ui.Text(c, text(v.Lang, copy.body)).TextColor(t.TextMuted)
	}
	actionRow(c, v, a)
	secondaryRow(c, v, a)
	updateSlot(c, v, a)
	riskLine(c, v, st)
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		ui.Text(c, text(v.Lang, steamWord(v.Status.SteamRunning))).Role(ui.RoleStatus)
		if stage, ok := app.Stage(v.Status.Busy); ok {
			ui.Text(c, text(v.Lang, mainStateKeys[stage])).Role(ui.RoleStatus)
		}
		if copy, ok := resultCopies[v.Status.Result]; ok {
			ui.Text(c, text(v.Lang, copy)).Role(ui.RoleStatus)
		}
	})
}

func riskLine(c *ui.Context, v app.View, st *State) {
	label, ok := RiskLabel(v.Lang, v.Health)
	if !v.Risk || !ok {
		return
	}
	t := c.Theme()
	line := ui.Button(c, label+"   "+SettingsAffordance(v.Lang)).TextColor(t.Warning)
	if line.Clicked() {
		st.Tab = int(app.TabSteam)
		st.SettingsOpen = true
	}
}

func compatibilitySection(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Text(c, text(v.Lang, compatSection)).FontSize(t.FontSize * 1.1)
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.Badge(c, HealthLabel(v.Lang, v.Health))
			if prewarmOffered(v) {
				ui.Row(c).Children(func() {
					if ui.Button(c, text(v.Lang, prewarmOffline)).Clicked() {
						a.Prewarm()
					}
				})
			}
			if tip, ok := HealthTip(v.Lang, v.Health); ok {
				ui.Text(c, tip).TextColor(t.TextMuted)
			}
		})
		switch {
		case v.Prewarming:
			ui.Text(c, text(v.Lang, cachingSignatures)).Role(ui.RoleStatus)
		case v.PrecacheReady:
			ui.Text(c, text(v.Lang, cacheReady)).Role(ui.RoleStatus)
		}
		if v.PrecacheFailure != app.NoResult {
			ui.Text(c, CacheFailure(v.Lang, v.PrecacheFailure)).TextColor(t.Danger)
		}
		ui.Collapsible(c, text(v.Lang, compatDetails), &st.CompatDetails, func() {
			compatibilityDetails(c, v, a)
		})
	})
}

func prewarmOffered(v app.View) bool {
	if v.Prewarming {
		return false
	}
	if v.Health == app.HealthAdaptedNotCached {
		return true
	}
	for _, component := range v.Components {
		if component.Health == app.HealthAdaptedNotCached {
			return true
		}
	}
	return false
}

func compatibilityDetails(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		for _, component := range v.Components {
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				ui.Text(c, ComponentLabel(v.Lang, component))
				if component.Fingerprint != "" {
					ui.Text(c, component.Fingerprint).TextColor(t.TextMuted)
				}
				ui.Badge(c, HealthLabel(v.Lang, component.Health))
			})
		}
		if prewarmOffered(v) {
			ui.Row(c).Children(func() {
				if ui.Button(c, text(v.Lang, cacheSignatures)).Clicked() {
					a.Prewarm()
				}
			})
		}
	})
}

func actionRow(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	busy := v.Busy
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		if ui.Button(c, text(v.Lang, applyAndStart)).Disabled(busy || v.PrimaryDisabled).Clicked() {
			a.Apply()
		}
		if v.Conflict == "" {
			return
		}
		ui.Text(c, conflictText(v.Lang, v.Conflict)).Role(ui.RoleStatus)
		if ui.Button(c, text(v.Lang, removeAndDeploy)).Disabled(busy).Clicked() {
			a.RemoveConflict()
		}
	})
}

var secondaryLabels = map[app.SecondaryAction]key{
	app.SecondaryUninstall:           uninstall,
	app.SecondaryExitAndUninstall:    exitAndUninstall,
	app.SecondaryUninstallAndRestart: uninstallAndRestart,
}

func secondaryRow(c *ui.Context, v app.View, a Actions) {
	if len(v.Secondary) == 0 {
		return
	}
	t := c.Theme()
	busy := v.Busy
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		for _, action := range v.Secondary {
			if ui.Button(c, text(v.Lang, secondaryLabels[action])).Disabled(busy).Clicked() {
				runSecondary(a, action)
			}
		}
	})
}

func runSecondary(a Actions, action app.SecondaryAction) {
	switch action {
	case app.SecondaryExitAndUninstall:
		a.ExitAndUninstall()
	case app.SecondaryUninstallAndRestart:
		a.UninstallAndRestart()
	default:
		a.Uninstall()
	}
}

var pendingQuestions = map[app.Confirmation]key{
	app.ConfirmExitRemove:       runningQuestion,
	app.ConfirmRemoveAndRestart: runningQuestion,
	app.ConfirmExit:             quitQuestion,
}

func confirm(c *ui.Context, v app.View, a Actions) {
	question, ok := pendingQuestions[v.Pending]
	if !ok {
		return
	}
	open := true
	switch ui.AlertDialog(c, &open, text(v.Lang, confirmTitle), text(v.Lang, question), text(v.Lang, yes), text(v.Lang, no)) {
	case 0:
		a.Confirm()
	case 1:
		a.Cancel()
	}
}

func updateSlot(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	progress := key("")
	if v.UpdateInFlight == app.BusyCheckUpdate {
		progress = checking
	}
	if v.UpdateInFlight == app.BusyDownloadPatch {
		progress = downloadingPatch
	}
	patchBusy := v.Status.Busy != app.Idle || v.UpdateInFlight != app.Idle
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		if ui.Button(c, text(v.Lang, checkUpdate)).Disabled(patchBusy).Clicked() {
			a.CheckUpdate()
		}
		if progress != "" {
			ui.Text(c, text(v.Lang, progress)).Role(ui.RoleStatus)
			return
		}
		if copy, ok := conclusionCopies[v.Update]; ok {
			ui.Text(c, text(v.Lang, copy)).Role(ui.RoleStatus)
		}
		if v.DownloadOffered {
			if ui.Button(c, text(v.Lang, downloadNewVersion)).Disabled(patchBusy).Clicked() {
				a.Download()
			}
		}
	})
}

func settingsDialog(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	panel := func() {
		ui.Column(c).Grow(1).MaxWidth(380).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
			ui.Text(c, text(v.Lang, settings)).FontSize(t.FontSize * 1.25)
			if ui.Tabs(c, &st.Tab, tabLabels(v.Lang)...).Changed() {
				if st.PathFocused {
					st.commitPath(a)
					st.PathFocused = false
				}
				if st.SourceFocused {
					st.commitSource()
					st.SourceFocused = false
				}
			}
			ui.Scroll(c).Grow(1).Children(func() {
				switch app.Tab(st.Tab) {
				case app.TabGeneral:
					generalPage(c, v, st, a)
				case app.TabPatchSource:
					patchSourcePage(c, v, st, a)
				case app.TabSteam:
					steamPage(c, v, st, a)
				case app.TabAbout:
					aboutPage(c, v, a)
				}
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c, text(v.Lang, closeButton)).Clicked() {
					st.SettingsOpen = false
				}
			})
		})
	}
	if ui.Modal(c, &st.SettingsOpen, panel).Dismissed() {
		st.SettingsOpen = true
	}
}

func aboutPage(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, text(v.Lang, appVersionLabel)).TextColor(t.TextMuted)
			ui.Text(c, v.Tool.Version)
		})
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, text(v.Lang, appUpdateRow)).TextColor(t.TextMuted)
			if ui.Button(c, text(v.Lang, checkToolUpdate)).Disabled(v.Busy).Clicked() {
				a.CheckToolUpdate()
			}
			toolStatus(c, v)
		})
		if v.Tool.State == app.ToolAvailable {
			if ui.Button(c, text(v.Lang, openDownloadPage)).Clicked() {
				a.OpenDownload()
			}
		}
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			if ui.Button(c, text(v.Lang, projectPage)).Clicked() {
				a.OpenProject()
			}
			if ui.Button(c, text(v.Lang, runWizard)).Clicked() {
				a.WizardRerun()
			}
		})
		ui.Text(c, text(v.Lang, runWizardHint)).TextColor(t.TextMuted)
	})
}

func toolStatus(c *ui.Context, v app.View) {
	t := c.Theme()
	switch v.Tool.State {
	case app.ToolChecking:
		ui.Text(c, text(v.Lang, toolChecking)).TextColor(t.TextMuted)
	case app.ToolUpToDate:
		ui.Text(c, text(v.Lang, toolUpToDate))
	case app.ToolAvailable:
		ui.Text(c, toolAvailableText(v.Lang, v.Tool.Available)).TextColor(t.TextMuted)
	case app.ToolCheckFailed:
		ui.Text(c, text(v.Lang, checkFailed)).TextColor(t.Danger)
	}
}

func generalPage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, text(v.Lang, languageRow))
			labels := choiceLabels(v.Lang, followSystem, languageChinese, languageEnglish)
			value := labels[st.Language]
			if ui.Select(c, &value, labels).Changed() {
				st.Language = slices.Index(labels, value)
				a.SetLanguage(app.Language(st.Language))
			}
		})
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, text(v.Lang, themeRow))
			labels := choiceLabels(v.Lang, followSystem, themeDark, themeLight)
			value := labels[st.Theme]
			if ui.Select(c, &value, labels).Changed() {
				st.Theme = slices.Index(labels, value)
				a.SetTheme(app.Theme(st.Theme))
			}
		})
	})
}

func pathField(c *ui.Context, v app.View, st *State, a Actions, label string, errorText key, description string) {
	submitted := false
	field := ui.Field(c, label, func() {
		input := ui.TextInput(c, &st.PathDraft).
			Label(label).
			Bind(&st.PathHandle)
		submitted = input.Submitted()
	})
	if description != "" {
		field.Description(description)
	}
	if st.PathError {
		field.Error(text(v.Lang, errorText))
	}
	if submitted {
		st.commitPath(a)
	}
	focused := st.PathHandle.Focused(c)
	if st.PathFocused && !focused && !submitted {
		st.commitPath(a)
	}
	st.PathFocused = focused
}

func steamPage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	pathField(c, v, st, a, text(v.Lang, steamPathRow), steamPathError, text(v.Lang, steamPathHint))
	if ui.Button(c, text(v.Lang, browse)).Clicked() {
		a.Browse()
	}
	entryControl(c, v, st, a)
	ui.Text(c, text(v.Lang, steamCfgNote)).TextColor(t.TextMuted)
	compatibilitySection(c, v, st, a)
}

func (st *State) commitPath(a Actions) {
	st.PathError = !a.CommitPath(st.PathDraft)
}

const CustomChoice = "custom"

func (st *State) Draft() source.Draft {
	if st.SourceChoice == CustomChoice {
		return source.Draft{Custom: strings.TrimSpace(st.SourceCustom)}
	}
	return source.Draft{Builtin: st.SourceChoice}
}

func (st *State) commitSource() {
	resolved, err := st.Draft().Resolve()
	if err != nil {
		st.SourceError = true
		return
	}
	st.showDraft(resolved)
}

func (st *State) SetSource(src source.Source) {
	st.Source = src
	st.showDraft(src)
}

func (st *State) showDraft(src source.Source) {
	st.SourceChoice = ChoiceOf(src)
	st.SourceCustom = ""
	if src.Custom {
		st.SourceCustom = src.Key
	}
	st.SourceError = false
}

func ChoiceOf(s source.Source) string {
	if s.Custom {
		return CustomChoice
	}
	return source.OrDefault(s).Key
}

func patchSourcePage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		sourceChoices(c, v, st)
		draft := st.Draft()
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			ui.Text(c, text(v.Lang, fileLayout)).TextColor(t.TextMuted)
			ui.Text(c, text(v.Lang, layoutLabels[draft.Layout()]))
		})
		if _, err := draft.Resolve(); err == nil && !draft.Same(st.Source) {
			if ui.Button(c, text(v.Lang, applyAndDownload)).Clicked() {
				if chosen, ok := a.ApplySource(draft); ok {
					st.SetSource(chosen)
					st.Entry = source.Effective(chosen, st.Entry)
				}
			}
		}
	})
}

func entryControl(c *ui.Context, v app.View, st *State, a Actions) {
	src := source.OrDefault(st.Source)
	if !src.Layout.Has(source.EntryEarly) {
		return
	}
	ui.Text(c, text(v.Lang, entryRow))
	ui.RadioGroup(c, func() {
		for _, entry := range []source.Entry{source.EntryDefault, source.EntryEarly} {
			radio := ui.Radio(c, &st.Entry, entry, text(v.Lang, entryLabels[entry]))
			radio.Disabled(src.EntryState(entry) != source.EntrySelectable)
			if radio.Changed() {
				a.SetEntry(entry)
			}
		}
	})
}
