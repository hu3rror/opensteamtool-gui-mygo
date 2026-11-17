package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestTheGateTakesOneKindAtATime(t *testing.T) {
	var gate app.Gate
	if !gate.Try(app.BusyApplyPatch) {
		t.Fatal("the gate refused the first kind")
	}
	if gate.Busy() != app.BusyApplyPatch {
		t.Errorf("the gate holds %v, want %v", gate.Busy(), app.BusyApplyPatch)
	}
	if gate.Try(app.BusyStartSteam) {
		t.Fatal("the gate granted a second kind")
	}
	if gate.Busy() != app.BusyApplyPatch {
		t.Errorf("a refused request changed the kind to %v", gate.Busy())
	}
	gate.Advance(app.BusyStartSteam)
	if gate.Busy() != app.BusyStartSteam {
		t.Errorf("the running kind did not advance: %v", gate.Busy())
	}
	gate.Release()
	if gate.Busy() != app.Idle {
		t.Errorf("the gate stayed %v after release", gate.Busy())
	}
	if !gate.Try(app.BusyStopSteam) {
		t.Fatal("the gate refused a kind after release")
	}
}

func TestTheGateRefusesToBeTakenForNothing(t *testing.T) {
	var gate app.Gate
	if gate.Try(app.Idle) {
		t.Fatal("the gate granted an idle kind")
	}
	if gate.Busy() != app.Idle {
		t.Errorf("the gate holds %v after refusing idle", gate.Busy())
	}
	gate.Release()
	gate.Advance(app.BusyStartSteam)
	if gate.Busy() != app.Idle {
		t.Errorf("an idle gate moved to %v", gate.Busy())
	}
}

func TestTheStatusBarCarriesTheSteamSlotInEverySnapshot(t *testing.T) {
	for _, running := range []bool{true, false} {
		for _, busy := range []app.Busy{app.Idle, app.BusyApplyPatch, app.BusyCheckUpdate} {
			view := app.Derive(app.Facts{SteamRunning: running, Busy: busy})
			if view.Status.SteamRunning != running {
				t.Errorf("the Steam slot reads %v with %v busy, want %v", view.Status.SteamRunning, busy, running)
			}
		}
	}
}

func TestTheStatusBarShowsTheStageOfNonUpdateKindsOnly(t *testing.T) {
	tests := []struct {
		busy app.Busy
		want app.Busy
	}{
		{app.Idle, app.Idle},
		{app.BusyApplyPatch, app.BusyApplyPatch},
		{app.BusyUninstall, app.BusyUninstall},
		{app.BusyStartSteam, app.BusyStartSteam},
		{app.BusyStopSteam, app.BusyStopSteam},
		{app.BusyCheckUpdate, app.Idle},
		{app.BusyDownloadPatch, app.Idle},
	}
	for _, test := range tests {
		if got := app.Derive(app.Facts{Busy: test.busy}).Status.Busy; got != test.want {
			t.Errorf("the stage slot with %v busy = %v, want %v", test.busy, got, test.want)
		}
	}
}

func TestTheStatusBarKeepsTheLastResult(t *testing.T) {
	for _, result := range []app.Result{app.NoResult, app.SteamStarted, app.SteamRestarted, app.MissingSteamProgram, app.CloseSteamFailed, app.StartSteamFailed} {
		if got := app.Derive(app.Facts{Result: result}).Status.Result; got != result {
			t.Errorf("the result slot reads %v, want %v", got, result)
		}
	}
}
