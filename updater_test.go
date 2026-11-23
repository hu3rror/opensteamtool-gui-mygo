package main

import (
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/rt"
)

func TestTheAboutButtonAsksForTheUpdateCheckOnlyWhenClicked(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	checks := 0
	s.updateCheck = func() { checks++ }

	if checks != 0 {
		t.Fatalf("the check ran %d times before a click", checks)
	}
	s.actions.CheckAppUpdate()
	if checks != 1 {
		t.Fatalf("the check ran %d times, want once", checks)
	}
	s.actions.CheckAppUpdate()
	if checks != 2 {
		t.Errorf("the check ran %d times, want a second check for the second click", checks)
	}
}

func TestBothSwitchesPassTheNewValueToTheUpdater(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	var checks, downloads []bool
	s.setChecks = func(on bool) { checks = append(checks, on) }
	s.setDownloads = func(on bool) { downloads = append(downloads, on) }

	s.actions.SetAutomaticChecks(true)
	s.actions.SetAutomaticDownloads(true)

	if !slices.Equal(checks, []bool{true}) {
		t.Errorf("checks = %v, want the one new value", checks)
	}
	if !slices.Equal(downloads, []bool{true}) {
		t.Errorf("downloads = %v, want the one new value", downloads)
	}
}

func TestAChangeNotificationRefreshesTheSnapshot(t *testing.T) {
	s := newTestSession(t, &rt.Fake{}, t.TempDir())
	available, checks, downloads := "", false, false
	s.updateAvailable = func() string { return available }
	s.checksOn = func() bool { return checks }
	s.downloadsOn = func() bool { return downloads }
	var change func()
	s.watchUpdates = func(fn func()) func() {
		change = fn
		return func() {}
	}

	s.followUpdater()
	if change == nil {
		t.Fatal("the session does not listen for a change")
	}
	available, checks, downloads = "0.6.0", true, true
	change()

	facts := s.snapshot()
	if facts.UpdateAvailable != "0.6.0" || !facts.AutomaticChecks || !facts.AutomaticDownloads {
		t.Errorf("facts = %+v, want the changed preferences and the available version", facts)
	}
}
