package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestTheThemeDecidesTheAppearanceItAsksFor(t *testing.T) {
	tests := []struct {
		name   string
		theme  app.Theme
		system bool
		known  bool
		dark   bool
	}{
		{name: "following a dark system", theme: app.ThemeSystem, system: true, known: true, dark: true},
		{name: "following a light system", theme: app.ThemeSystem, system: false, known: true},
		{name: "following an unknown system is dark", theme: app.ThemeSystem, system: false, dark: true},
		{name: "dark beats a light system", theme: app.ThemeDark, system: false, known: true, dark: true},
		{name: "dark beats an unknown system", theme: app.ThemeDark, dark: true},
		{name: "light beats a dark system", theme: app.ThemeLight, system: true, known: true},
		{name: "light beats an unknown system", theme: app.ThemeLight},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.theme.Dark(test.system, test.known); got != test.dark {
				t.Errorf("dark = %v, want %v", got, test.dark)
			}
		})
	}
}
