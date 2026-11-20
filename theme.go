package main

import (
	"github.com/egoist/mygo"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
)

func applyAppearance(dark bool) {
	source := mygo.ThemeLight
	if dark {
		source = mygo.ThemeDark
	}
	mygo.Theme.SetSource(source)
}

func (s *session) theme() app.Theme {
	facts := s.factsWithoutGate()
	if facts.Wizard.Open {
		return facts.Wizard.Theme
	}
	return s.config().Theme
}

func (s *session) applyTheme() {
	systemDark, systemKnown := s.systemDark()
	s.applyDark(s.theme().Dark(systemDark, systemKnown))
}

func (s *session) setTheme(theme app.Theme) {
	if theme == s.config().Theme {
		return
	}
	s.writeConfig(func(cfg *config.Config) { cfg.Theme = theme })
	s.applyTheme()
}

func (s *session) followSystem() {
	s.stopTheme = s.watchTheme(func() { s.onUI(s.applyTheme) })
}
