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
}

type Actions struct {
	SetLanguage func(app.Language)
	CommitPath  func(raw string) bool
	Browse      func()
	ApplySource func(draft source.Draft) (source.Source, bool)
	SetEntry    func(source.Entry)
}

func Main(c *ui.Context, v app.View, st *State, a Actions) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(2)).Children(func() {
		if v.NeedsSetup {
			setup(c, v, st)
			return
		}
		mainScreen(c, v, st)
	})
	if st.SettingsOpen {
		settingsDialog(c, v, st, a)
	}
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

func mainScreen(c *ui.Context, v app.View, st *State) {
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
