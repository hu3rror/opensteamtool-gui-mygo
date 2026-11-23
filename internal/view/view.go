package view

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yldm-tech/mygo-breeze-ui/ui/breeze"

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
	PathNotFound bool
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
	Detect              func()
	OpenLua             func()
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
	breeze.Use(c)
	t := c.Theme()
	ui.Column(c).Fill().Children(func() {
		titleBar(c, v)
		ui.Column(c).FillWidth().Grow(1).Padding(t.Space(6)).Gap(t.Space(2)).Children(func() {
			if v.Wizard.Open {
				wizard(c, v, st, a)
				return
			}
			mainScreen(c, v, st, a)
		})
	})
	if st.SettingsOpen {
		settingsDialog(c, v, st, a)
	}
	confirm(c, v, a)
}

func titleBar(c *ui.Context, v app.View) {
	if c.TitleBar().Height <= 0 {
		return
	}
	breeze.HeaderBar(c, Title(), v.Tool.Version, nil, nil)
}

func wizard(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Children(func() {
		breeze.Text(c, text(v.Lang, wizardTitle)).TextColor(t.TextMuted)
		ui.Spacer(c)
		breeze.Text(c, WizardStepLabel(v.Lang, v.Wizard.Step)).TextColor(t.TextMuted)
	})
	breeze.Text(c, text(v.Lang, wizardStepNames[v.Wizard.Step]), breeze.TextSubtitle).Role(ui.RoleStatus)
	breeze.Text(c, text(v.Lang, wizardStepHints[v.Wizard.Step])).TextColor(t.TextMuted)
	breeze.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			switch v.Wizard.Step {
			case app.WizardLanguage:
				wizardLanguage(c, v, st, a)
			case app.WizardTheme:
				themeCards(c, v, st, a)
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
	if breeze.ComboRow(c, text(v.Lang, languageRow), "", &value, labels).Changed() {
		a.WizardLanguage(app.Language(slices.Index(labels, value)))
	}
}

func themeCards(c *ui.Context, v app.View, st *State, a Actions) {
	platform := breeze.PlatformOf(c)
	window := breeze.Theme(c, platform)
	dark := breeze.ThemeOf(c, platform, true, ui.Color{})
	light := breeze.ThemeOf(c, platform, false, ui.Color{})
	height := window.Space(28)
	preview := func(look *ui.Theme) {
		ui.Themed(c, look).Fill().Children(func() {
			ui.Column(c).Grow(1).Fill().Background(look.Background).Radius(look.Radius).Padding(look.Space(2)).Gap(look.Space(1)).Children(func() {
				ui.Box(c).Height(look.Space(1)).Background(look.Background.Mix(look.Text, barTint))
				ui.Box(c).Grow(1).Fill().Background(look.Background.Mix(look.Text, fillTint)).Radius(look.Radius)
			})
		})
	}
	card := func(theme app.Theme, label key, looks []*ui.Theme) {
		box := ui.RadioBase(c, &st.WizardTheme, int(theme))
		if box.Changed() {
			a.WizardTheme(theme)
		}
		border, width := window.Border, float32(1)
		if st.WizardTheme == int(theme) {
			border, width = window.Accent, 2
		}
		box.Grow(1).Height(height).Border(width, border).Radius(window.Radius).Children(func() {
			ui.Column(c).Fill().Children(func() {
				ui.Column(c).Grow(1).Fill().Children(func() {
					if len(looks) == 1 {
						preview(looks[0])
						return
					}
					ui.Row(c).Fill().Children(func() {
						for _, look := range looks {
							ui.Column(c).Grow(1).Fill().Children(func() { preview(look) })
						}
					})
				})
				ui.Column(c).Padding(window.Space(1)).Background(window.Background).Children(func() {
					breeze.Text(c, text(v.Lang, label)).TextColor(window.Text)
				})
			})
		})
	}
	breeze.RadioGroup(c, func() {
		ui.Column(c).Background(window.Surface).Radius(window.Radius).Padding(window.Space(2)).Children(func() {
			ui.Row(c).Height(height).Gap(window.Space(2)).Children(func() {
				card(app.ThemeSystem, followSystem, []*ui.Theme{light, dark})
				card(app.ThemeDark, themeDark, []*ui.Theme{dark})
				card(app.ThemeLight, themeLight, []*ui.Theme{light})
			})
		})
	}).Label(text(v.Lang, themeRow))
}

const (
	fillTint = 0.16
	barTint  = 0.3
)

func wizardFooter(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	ui.Row(c).Gap(t.Space(2)).Children(func() {
		wizardBack(c, v, a)
		if v.Wizard.Step == app.WizardDownload && v.Wizard.Download != app.WizardDownloaded {
			if breeze.Button(c, text(v.Lang, wizardSkip)).Clicked() {
				a.WizardSkip()
			}
		}
		ui.Spacer(c)
		switch v.Wizard.Step {
		case app.WizardDownload:
			switch v.Wizard.Download {
			case app.WizardNotDownloaded:
				if breeze.Button(c, text(v.Lang, wizardDownload)).Clicked() {
					a.WizardDownload()
				}
			case app.WizardDownloaded:
				if breeze.Button(c, text(v.Lang, wizardFinish)).Clicked() {
					a.WizardFinish()
				}
			case app.WizardDownloadFailed:
				if breeze.Button(c, text(v.Lang, wizardRetry)).Clicked() {
					a.WizardDownload()
				}
			}
		default:
			if breeze.Button(c, text(v.Lang, nextStep)).Disabled(v.Wizard.NextDisabled).Clicked() {
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
	if breeze.Button(c, text(v.Lang, backStep)).Clicked() {
		a.WizardBack()
	}
}

func wizardPath(c *ui.Context, v app.View, st *State, a Actions) {
	breeze.Card(c, func() {
		pathRow(c, v, st, a, text(v.Lang, steamPathRow), wizardPathError, "")
	})
	sourceChoices(c, v, st)
}

func sourceChoices(c *ui.Context, v app.View, st *State) {
	breeze.Card(c, func() {
		breeze.RadioGroup(c, func() {
			choice := func(name, label, note string) {
				row := breeze.RadioRow(c, &st.SourceChoice, name, label, "", nil)
				if note != "" {
					row.Tooltip(note)
				}
				if row.Changed() {
					st.SourceError = false
				}
			}
			for _, builtin := range source.Builtins() {
				choice(builtin.Key, builtin.Repository, sourceTooltip(v.Lang, builtin))
			}
			choice(CustomChoice, text(v.Lang, customRepository), "")
		})
	})
	t := c.Theme()
	if st.SourceChoice == source.KeyOfficial {
		outsideCard(c, 0, func() {
			breeze.Text(c, text(v.Lang, officialStale)).TextColor(t.Warning)
		})
	}
	if st.SourceChoice != CustomChoice {
		return
	}
	submitted := false
	field := breeze.Field(c, text(v.Lang, customSourceHint), func() {
		input := breeze.TextInput(c, &st.SourceCustom).Bind(&st.SourceHandle)
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
	outsideCard(c, 0, func() {
		switch w.Download {
		case app.WizardNotDownloaded:
			breeze.Text(c, text(v.Lang, patchNotDownloaded))
		case app.WizardDownloaded:
			breeze.Text(c, text(v.Lang, wizardDownloaded))
		case app.WizardDownloading:
			breeze.Text(c, text(v.Lang, wizardDownloading))
		case app.WizardDownloadFailed:
			breeze.Text(c, WizardDownloadFailure(v.Lang, w.Failure)).Role(ui.RoleStatus).TextColor(t.Warning)
		}
	})
}

func mainScreen(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(t.Space(4)).Children(func() {
		breeze.Text(c, text(v.Lang, mainStateKeys[v.MainState]), breeze.TextTitle).Role(ui.RoleStatus)
		ui.Spacer(c)
		if breeze.Button(c, text(v.Lang, settings)).Clicked() {
			st.SettingsOpen = true
		}
	})
	if copy, ok := explanationCopies[v.Explanation]; ok {
		breeze.Text(c, text(v.Lang, copy.title), breeze.TextHeading)
		breeze.Text(c, text(v.Lang, copy.body)).TextColor(t.TextMuted)
	}
	actionRow(c, v, a)
	secondaryRow(c, v, a)
	updateSlot(c, v, a)
	riskLine(c, v, st)
	ui.Row(c).Gap(t.Space(4)).Wrap().Children(func() {
		breeze.Text(c, text(v.Lang, steamWord(v.Status.SteamRunning))).Role(ui.RoleStatus)
		if stage, ok := app.Stage(v.Status.Busy); ok {
			breeze.Text(c, text(v.Lang, mainStateKeys[stage])).Role(ui.RoleStatus)
		}
		if copy, ok := resultCopies[v.Status.Result]; ok {
			breeze.Text(c, text(v.Lang, copy)).Role(ui.RoleStatus)
		}
	})
}

func riskLine(c *ui.Context, v app.View, st *State) {
	label, ok := RiskLabel(v.Lang, v.Health)
	if !v.Risk || !ok {
		return
	}
	t := c.Theme()
	line := breeze.Button(c, label+"   "+SettingsAffordance(v.Lang)).TextColor(t.Warning)
	if line.Clicked() {
		st.Tab = int(app.TabSteam)
		st.SettingsOpen = true
	}
}

func compatibilitySection(c *ui.Context, v app.View, st *State, a Actions) {
	breeze.Card(c, func() {
		compatHead(c, v, a)
		breeze.ExpanderRow(c, text(v.Lang, compatDetails), "", &st.CompatDetails, func() {
			compatibilityDetails(c, v, a)
		})
	})
}

func compatHead(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	tip, _ := HealthTip(v.Lang, v.Health)
	ui.Column(c).Padding(t.Space(4), t.Space(4), 0, t.Space(4)).Gap(t.Space(2)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			noteWithInfo(c, func() { breeze.Text(c, text(v.Lang, compatSection)) }, tip)
			ui.Spacer(c)
			breeze.Badge(c, HealthLabel(v.Lang, v.Health))
		})
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			if prewarmOffered(v) {
				if breeze.Button(c, text(v.Lang, prewarmOffline)).Clicked() {
					a.Prewarm()
				}
			}
			ui.Spacer(c)
		})
		switch {
		case v.Prewarming:
			breeze.Text(c, text(v.Lang, cachingSignatures)).Role(ui.RoleStatus)
		case v.PrecacheReady:
			breeze.Text(c, text(v.Lang, cacheReady)).Role(ui.RoleStatus)
		}
		if v.PrecacheFailure != app.NoResult {
			breeze.Text(c, CacheFailure(v.Lang, v.PrecacheFailure)).TextColor(t.Danger)
		}
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
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Row(c).Gap(t.Space(2)).Children(func() {
					breeze.Text(c, ComponentLabel(v.Lang, component))
					ui.Spacer(c)
					breeze.Badge(c, HealthLabel(v.Lang, component.Health))
				})
				if component.Fingerprint != "" {
					breeze.Text(c, component.Fingerprint).TextColor(t.TextMuted)
				}
			})
		}
		if prewarmOffered(v) {
			ui.Row(c).Children(func() {
				if breeze.Button(c, text(v.Lang, cacheSignatures)).Clicked() {
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
		if breeze.PrimaryButton(c, text(v.Lang, applyAndStart)).Disabled(busy || v.PrimaryDisabled).Clicked() {
			a.Apply()
		}
		if v.Conflict == "" {
			return
		}
		breeze.Text(c, conflictText(v.Lang, v.Conflict)).Role(ui.RoleStatus)
		if breeze.Button(c, text(v.Lang, removeAndDeploy)).Disabled(busy).Clicked() {
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
			if breeze.Button(c, text(v.Lang, secondaryLabels[action])).Disabled(busy).Clicked() {
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

type pendingQuestion struct {
	text        key
	destructive bool
}

var pendingQuestions = map[app.Confirmation]pendingQuestion{
	app.ConfirmExitRemove:       {text: runningQuestion, destructive: true},
	app.ConfirmRemoveAndRestart: {text: runningQuestion, destructive: true},
	app.ConfirmExit:             {text: quitQuestion},
}

const (
	yesChoice = iota
	noChoice
)

func confirm(c *ui.Context, v app.View, a Actions) {
	pending, ok := pendingQuestions[v.Pending]
	if !ok {
		return
	}
	open := true
	title, message := text(v.Lang, confirmTitle), text(v.Lang, pending.text)
	yesButton, noButton := text(v.Lang, yes), text(v.Lang, no)
	var chosen int
	if pending.destructive {
		chosen = breeze.DestructiveAlertDialog(c, &open, title, message, yesChoice, yesButton, noButton)
	} else {
		chosen = breeze.AlertDialog(c, &open, title, message, yesButton, noButton)
	}
	switch chosen {
	case yesChoice:
		a.Confirm()
	case noChoice:
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
	ui.Row(c).Gap(t.Space(4)).Wrap().Children(func() {
		if breeze.Button(c, text(v.Lang, checkUpdate)).Disabled(patchBusy).Clicked() {
			a.CheckUpdate()
		}
		if progress != "" {
			breeze.Text(c, text(v.Lang, progress)).Role(ui.RoleStatus)
			return
		}
		if copy, ok := conclusionCopies[v.Update]; ok {
			breeze.Text(c, text(v.Lang, copy)).Role(ui.RoleStatus)
		}
		if v.DownloadOffered {
			if breeze.Button(c, text(v.Lang, downloadNewVersion)).Disabled(patchBusy).Clicked() {
				a.Download()
			}
		}
	})
}

const tabsWidth float32 = 420

const infoSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16">
  <circle cx="8" cy="8" r="6.6" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <rect x="7.2" y="6.6" width="1.6" height="4.8" rx="0.8" fill="currentColor"/>
  <circle cx="8" cy="4.5" r="1" fill="currentColor"/>
</svg>`

var infoIcon = ui.MustParseSVG([]byte(infoSVG))

func infoTip(c *ui.Context, tip string) ui.Element {
	return breeze.Tooltip(c, ui.Icon(c, infoIcon).Label(tip).Focusable(), tip)
}

func noteWithInfo(c *ui.Context, label func(), tip string) {
	ui.Row(c).Gap(c.Theme().Space(2)).Children(func() {
		label()
		if tip == "" {
			return
		}
		infoTip(c, tip)
	})
}

func changeTab(st *State, a Actions) {
	if st.PathFocused {
		st.commitPath(a)
		st.PathFocused = false
	}
	if st.SourceFocused {
		st.commitSource()
		st.SourceFocused = false
	}
}

func settingsDialog(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	panel := func() {
		column := ui.Column(c).Grow(1).MaxWidth(380).Gap(t.Space(3))
		if v.WindowWidth > 0 {
			column.Width(min(380, v.WindowWidth-40))
		}
		column.Children(func() {
			breeze.Text(c, text(v.Lang, settings), breeze.TextTitle)
			labels := tabLabels(v.Lang)
			if v.WindowWidth > 0 && v.WindowWidth < tabsWidth {
				value := labels[st.Tab]
				if breeze.Select(c, &value, labels).Changed() {
					st.Tab = slices.Index(labels, value)
					changeTab(st, a)
				}
			} else if breeze.Tabs(c, &st.Tab, labels...).Changed() {
				changeTab(st, a)
			}
			breeze.Scroll(c).Grow(1).Children(func() {
				switch app.Tab(st.Tab) {
				case app.TabGeneral:
					generalPage(c, v, st, a)
				case app.TabPatchSource:
					patchSourcePage(c, v, st, a)
				case app.TabSteam:
					steamPage(c, v, st, a)
				case app.TabLua:
					luaPage(c, v, a)
				case app.TabAbout:
					aboutPage(c, v, a)
				}
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if breeze.Button(c, text(v.Lang, closeButton)).Clicked() {
					st.SettingsOpen = false
				}
			})
		})
	}
	if breeze.Modal(c, &st.SettingsOpen, panel).Dismissed() {
		st.SettingsOpen = true
	}
}

func aboutPage(c *ui.Context, v app.View, a Actions) {
	breeze.Card(c, func() {
		breeze.ActionRow(c, text(v.Lang, appVersionLabel), "", func() {
			breeze.Text(c, v.Tool.Version)
		})
		breeze.ActionRow(c, text(v.Lang, appUpdateRow), "", func() {
			if breeze.Button(c, text(v.Lang, checkToolUpdate)).Disabled(v.Busy).Clicked() {
				a.CheckToolUpdate()
			}
			toolStatus(c, v)
		})
		if v.Tool.State == app.ToolAvailable {
			breeze.ActionRow(c, text(v.Lang, openDownloadPage), "", func() {
				if breeze.Button(c, text(v.Lang, openDownloadPage)).Clicked() {
					a.OpenDownload()
				}
			})
		}
		breeze.ActionRow(c, text(v.Lang, projectPage), "", func() {
			if breeze.Button(c, text(v.Lang, projectPage)).Clicked() {
				a.OpenProject()
			}
		})
		breeze.ActionRow(c, text(v.Lang, runWizard), text(v.Lang, runWizardHint), func() {
			if breeze.Button(c, text(v.Lang, runWizard)).Clicked() {
				a.WizardRerun()
			}
		})
	})
}

func toolStatus(c *ui.Context, v app.View) {
	t := c.Theme()
	switch v.Tool.State {
	case app.ToolChecking:
		breeze.Text(c, text(v.Lang, toolChecking)).TextColor(t.TextMuted)
	case app.ToolUpToDate:
		breeze.Text(c, text(v.Lang, toolUpToDate))
	case app.ToolAvailable:
		breeze.Text(c, toolAvailableText(v.Lang, v.Tool.Available)).TextColor(t.TextMuted)
	case app.ToolCheckFailed:
		breeze.Text(c, text(v.Lang, checkFailed)).TextColor(t.Danger)
	}
}

func generalPage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		breeze.Card(c, func() {
			labels := choiceLabels(v.Lang, followSystem, languageChinese, languageEnglish)
			value := labels[st.Language]
			if breeze.ComboRow(c, text(v.Lang, languageRow), "", &value, labels).Changed() {
				st.Language = slices.Index(labels, value)
				a.SetLanguage(app.Language(st.Language))
			}
			themes := choiceLabels(v.Lang, followSystem, themeDark, themeLight)
			chosen := themes[st.Theme]
			if breeze.ComboRow(c, text(v.Lang, themeRow), "", &chosen, themes).Changed() {
				st.Theme = slices.Index(themes, chosen)
				a.SetTheme(app.Theme(st.Theme))
			}
		})
	})
}

func pathRow(c *ui.Context, v app.View, st *State, a Actions, label string, errorText key, description string) {
	t := c.Theme()
	breeze.ActionRow(c, label, description, func() {
		submitted := false
		ui.Column(c).Grow(1).Gap(t.Space(1.5)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).Children(func() {
				input := breeze.TextInput(c, &st.PathDraft).
					Label(label).
					Bind(&st.PathHandle)
				submitted = input.Submitted()
				if st.PathError {
					input.Error(text(v.Lang, errorText))
				}
				if breeze.Button(c, text(v.Lang, browse)).Clicked() {
					a.Browse()
				}
				if breeze.Button(c, text(v.Lang, autoDetect)).Clicked() {
					a.Detect()
				}
				if v.SteamCfg {
					infoTip(c, text(v.Lang, steamCfgNote))
				}
			})
			if st.PathError {
				breeze.Text(c, text(v.Lang, errorText)).TextColor(t.Danger)
			}
			if st.PathNotFound {
				breeze.Text(c, text(v.Lang, steamNotFound)).TextColor(t.Warning)
			}
		})
		if submitted {
			st.commitPath(a)
		}
		focused := st.PathHandle.Focused(c)
		if st.PathFocused && !focused && !submitted {
			st.commitPath(a)
		}
		st.PathFocused = focused
	})
}

func steamPage(c *ui.Context, v app.View, st *State, a Actions) {
	ui.Column(c).Gap(c.Theme().Space(4)).Children(func() {
		breeze.Card(c, func() {
			pathRow(c, v, st, a, text(v.Lang, steamPathRow), steamPathError, text(v.Lang, steamPathHint))
		})
		entryControl(c, v, st, a)
		compatibilitySection(c, v, st, a)
	})
}

func outsideCard(c *ui.Context, top float32, fn func()) {
	t := c.Theme()
	ui.Column(c).PaddingStart(t.Space(4)).Margin(t.Space(top), 0, 0, 0).Children(fn)
}
func luaPage(c *ui.Context, v app.View, a Actions) {
	t := c.Theme()
	if v.LuaDir == "" {
		outsideCard(c, 0, func() {
			breeze.Text(c, text(v.Lang, pathUnset)).TextColor(t.TextMuted)
		})
		return
	}
	ui.Column(c).Gap(t.Space(4)).Children(func() {
		breeze.Card(c, func() {
			breeze.ActionRow(c, text(v.Lang, luaLayoutRow), "", func() {
				breeze.Text(c, v.LuaRelative)
			})
			breeze.ActionRow(c, text(v.Lang, luaPathRow), "", func() {
				breeze.Text(c, v.LuaDir)
				ui.Spacer(c)
				infoTip(c, LuaNote(v.Lang, v.LuaManifest))
			})
		})
		breeze.ActionRow(c, text(v.Lang, openLuaFolder), "", func() {
			if breeze.Button(c, text(v.Lang, openLuaFolder)).Clicked() {
				a.OpenLua()
			}
		})
	})
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
	sourceChoices(c, v, st)
	draft := st.Draft()
	if _, err := draft.Resolve(); err != nil || draft.Same(st.Source) {
		return
	}
	outsideCard(c, 3, func() {
		if breeze.Button(c, text(v.Lang, applyAndDownload)).Clicked() {
			if chosen, ok := a.ApplySource(draft); ok {
				st.SetSource(chosen)
				st.Entry = source.Effective(chosen, st.Entry)
			}
		}
	})
}

func entryControl(c *ui.Context, v app.View, st *State, a Actions) {
	src := source.OrDefault(st.Source)
	if !src.Layout.Has(source.EntryEarly) {
		return
	}
	breeze.Card(c, func() {
		ui.Column(c).Padding(c.Theme().Space(4), c.Theme().Space(4), 0, c.Theme().Space(4)).Children(func() {
			noteWithInfo(c, func() { breeze.Text(c, text(v.Lang, entryRow)) }, text(v.Lang, entryChoiceHint))
		})
		breeze.RadioGroup(c, func() {
			for _, entry := range []source.Entry{source.EntryDefault, source.EntryEarly} {
				row := breeze.RadioRow(c, &st.Entry, entry, text(v.Lang, entryLabels[entry]), "", nil)
				row.Disabled(src.EntryState(entry) != source.EntrySelectable)
				if row.Changed() {
					a.SetEntry(entry)
				}
			}
		})
	})
}
