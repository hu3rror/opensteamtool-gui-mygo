package view

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/source"
)

type State struct {
	SettingsOpen bool
	Tab          int
	Language     int
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
}

type Actions struct {
	SetLanguage         func(app.Language)
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
}

func Main(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(2)).Children(func() {
		if v.NeedsSetup {
			setup(c, v, st)
			return
		}
		mainScreen(c, v, st, a)
	})
	if st.SettingsOpen {
		settingsDialog(c, v, st, a)
	}
	confirm(c, v, a)
}

func setup(c *ui.Context, v app.View, st *State) {
	t := c.Theme()
	ui.Text(c, text(v.Lang, setupNotice)).Role(ui.RoleStatus)
	ui.Text(c, text(v.Lang, setupHint)).TextColor(t.TextMuted)
	if ui.Button(c, text(v.Lang, openSettings)).Clicked() {
		st.Tab = int(app.TabSteam)
		st.SettingsOpen = true
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
	busy := v.Status.Busy != app.Idle || v.UpdateInFlight != app.Idle
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
	busy := v.Status.Busy != app.Idle || v.UpdateInFlight != app.Idle
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
	busy := v.Status.Busy != app.Idle
	inflight := v.UpdateInFlight != app.Idle
	progress := key("")
	if v.UpdateInFlight == app.BusyCheckUpdate {
		progress = checking
	}
	if v.UpdateInFlight == app.BusyDownloadPatch {
		progress = downloadingPatch
	}
	ui.Row(c).Gap(t.Space(4)).Children(func() {
		if ui.Button(c, text(v.Lang, checkUpdate)).Disabled(busy || inflight).Clicked() {
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
			if ui.Button(c, text(v.Lang, downloadNewVersion)).Disabled(busy).Clicked() {
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

func generalPage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(2)).Children(func() {
		ui.Text(c, text(v.Lang, languageRow))
		choice := ui.Segmented(c, &st.Language,
			text(v.Lang, languageAuto),
			text(v.Lang, languageChinese),
			text(v.Lang, languageEnglish),
		)
		if choice.Changed() {
			a.SetLanguage(app.Language(st.Language))
		}
	})
}

func steamPage(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	submitted := false
	label := text(v.Lang, steamPathRow)
	field := ui.Field(c, label, func() {
		input := ui.TextInput(c, &st.PathDraft).
			Label(label).
			Bind(&st.PathHandle)
		submitted = input.Submitted()
	}).Description(text(v.Lang, steamPathHint))
	if st.PathError {
		field.Error(text(v.Lang, steamPathError))
	}
	if submitted {
		st.commitPath(a)
	}
	focused := st.PathHandle.Focused(c)
	if st.PathFocused && !focused && !submitted {
		st.commitPath(a)
	}
	st.PathFocused = focused
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

func (st *State) draft() source.Draft {
	if st.SourceChoice == CustomChoice {
		return source.Draft{Custom: strings.TrimSpace(st.SourceCustom)}
	}
	return source.Draft{Builtin: st.SourceChoice}
}

func (st *State) commitSource() {
	resolved, err := st.draft().Resolve()
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
		if st.SourceChoice == CustomChoice {
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
		draft := st.draft()
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
