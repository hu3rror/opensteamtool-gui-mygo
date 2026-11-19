package main

import (
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
)

func darkRecorder(applied *[]bool) func(bool) {
	return func(dark bool) { *applied = append(*applied, dark) }
}

func TestTheThemeAppliesAtOnceAndIsRemembered(t *testing.T) {
	dir := t.TempDir()
	host := &rt.Fake{}
	s := newSession(host, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto}, true, t.TempDir())
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	var applied []bool
	s.applyDark = darkRecorder(&applied)
	s.systemDark = func() (bool, bool) { return false, true }

	s.setTheme(app.ThemeDark)
	if !slices.Equal(applied, []bool{true}) {
		t.Fatalf("applied %v, want the dark appearance once", applied)
	}
	if facts := s.factsWithoutGate(); facts.Theme != app.ThemeDark {
		t.Fatalf("facts carry theme %v, want the dark one", facts.Theme)
	}
	s.syncSettings(s.factsWithoutGate())
	if s.uiState.Theme != int(app.ThemeDark) {
		t.Fatalf("the settings show theme %d, want the dark one", s.uiState.Theme)
	}
	cfg, readable, err := config.Load(dir, host)
	if err != nil || !readable {
		t.Fatalf("Load: %v, readable %v", err, readable)
	}
	if cfg.Theme != app.ThemeDark {
		t.Fatalf("saved theme is %v, want the dark one", cfg.Theme)
	}

	s.setTheme(app.ThemeDark)
	if len(applied) != 1 {
		t.Errorf("applied %v, an unchanged choice must apply nothing", applied)
	}

	restarted := newSession(host, noSleep, "zh-CN", cfg, readable, t.TempDir())
	var reapplied []bool
	restarted.applyDark = darkRecorder(&reapplied)
	restarted.systemDark = func() (bool, bool) { return false, true }
	restarted.applyTheme()
	if !slices.Equal(reapplied, []bool{true}) {
		t.Errorf("the restarted app applied %v, want the saved dark appearance", reapplied)
	}
}

func TestFollowingTheSystemAppliesItsAppearanceAndItsChanges(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto, Theme: app.ThemeSystem}, true, t.TempDir())
	var applied []bool
	s.applyDark = darkRecorder(&applied)
	systemDark := false
	s.systemDark = func() (bool, bool) { return systemDark, true }
	var watched []func()
	s.watchTheme = func(fn func()) func() {
		watched = append(watched, fn)
		return func() {}
	}

	s.followSystem()
	if len(watched) != 1 {
		t.Fatalf("registered %d listeners, want one", len(watched))
	}
	s.applyTheme()
	if !slices.Equal(applied, []bool{false}) {
		t.Fatalf("applied %v, want the light system appearance", applied)
	}

	systemDark = true
	watched[0]()
	if !slices.Equal(applied, []bool{false, true}) {
		t.Errorf("applied %v, the system change did not reach the appearance", applied)
	}
}

func TestAnUnknownSystemAppearanceIsAppliedAsDark(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto, Theme: app.ThemeDark}, true, t.TempDir())
	var applied []bool
	s.applyDark = darkRecorder(&applied)
	s.systemDark = func() (bool, bool) { return false, false }

	s.setTheme(app.ThemeSystem)
	if !slices.Equal(applied, []bool{true}) {
		t.Fatalf("applied %v, an unknown system appearance must be dark", applied)
	}

	s.setTheme(app.ThemeLight)
	if !slices.Equal(applied, []bool{true, false}) {
		t.Errorf("applied %v, an explicit light theme must ignore the system", applied)
	}
}

func TestTheSystemListenerIsLetGoWhenTheAppLeaves(t *testing.T) {
	s := newSession(&rt.Fake{}, noSleep, "zh-CN", config.Config{Language: app.LanguageAuto, Theme: app.ThemeSystem}, true, t.TempDir())
	released := 0
	s.watchTheme = func(func()) func() { return func() { released = released + 1 } }

	s.followSystem()
	s.exit()

	if released != 1 {
		t.Errorf("released the listener %d times, want once when the app leaves", released)
	}
}
