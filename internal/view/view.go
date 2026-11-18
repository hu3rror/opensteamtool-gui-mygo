package view

import (
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
)

type State struct {
	SettingsOpen bool
	Tab          int
	Language     int
	PathDraft    string
	PathFocused  bool
	PathError    bool
	PathHandle   ui.Handle
}

type Actions struct {
	SetLanguage func(app.Language)
	CommitPath  func(raw string) bool
	Browse      func()
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
			if ui.Tabs(c, &st.Tab, tabLabels(v.Lang)...).Changed() && st.PathFocused {
				st.commitPath(a)
				st.PathFocused = false
			}
			ui.Scroll(c).Grow(1).Children(func() {
				switch app.Tab(st.Tab) {
				case app.TabGeneral:
					generalPage(c, v, st, a)
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
	ui.Text(c, text(v.Lang, steamCfgNote)).TextColor(t.TextMuted)
}

func (st *State) commitPath(a Actions) {
	st.PathError = !a.CommitPath(st.PathDraft)
}
