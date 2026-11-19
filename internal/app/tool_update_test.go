package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestTheSelfUpdateCheckIsItsOwnChain(t *testing.T) {
	tests := []struct {
		name      string
		facts     app.Facts
		state     app.ToolState
		available string
	}{
		{
			name:  "nothing is said before a check",
			facts: app.Facts{ToolVersion: "0.1.0"},
			state: app.ToolUnchecked,
		},
		{
			name:  "the check in flight is said",
			facts: app.Facts{ToolVersion: "0.1.0", Busy: app.BusyCheckTool},
			state: app.ToolChecking,
		},
		{
			name:  "an equal release is up to date",
			facts: app.Facts{ToolVersion: "0.1.0", ToolChecked: true, ToolOnline: "v0.1.0"},
			state: app.ToolUpToDate,
		},
		{
			name:      "a newer release is offered",
			facts:     app.Facts{ToolVersion: "0.1.0", ToolChecked: true, ToolOnline: "0.2.0"},
			state:     app.ToolAvailable,
			available: "0.2.0",
		},
		{
			name:  "an older release is not offered",
			facts: app.Facts{ToolVersion: "0.2.0", ToolChecked: true, ToolOnline: "0.1.0"},
			state: app.ToolUpToDate,
		},
		{
			name:  "a failed check says so",
			facts: app.Facts{ToolVersion: "0.1.0", ToolFailed: true},
			state: app.ToolCheckFailed,
		},
		{
			name:  "a later check waits for its own result",
			facts: app.Facts{ToolVersion: "0.1.0", ToolFailed: true, Busy: app.BusyCheckTool},
			state: app.ToolChecking,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := app.Derive(test.facts).Tool
			if tool.Version != test.facts.ToolVersion {
				t.Errorf("version = %q, want %q", tool.Version, test.facts.ToolVersion)
			}
			if tool.State != test.state || tool.Available != test.available {
				t.Errorf("check = %+v, want state %v with %q", tool, test.state, test.available)
			}
		})
	}
}

func TestTheToolCheckAndThePatchCheckNeverSeeEachOther(t *testing.T) {
	toolBusy := app.Derive(app.Facts{
		ToolVersion:     "0.1.0",
		PatchDownloaded: true,
		PatchComplete:   true,
		PatchVersion:    "v1",
		OnlineVersion:   "v1",
		UpdateChecked:   true,
		Busy:            app.BusyCheckTool,
		ToolChecked:     true,
		ToolOnline:      "9.9.9",
	})
	if toolBusy.Update != app.UpToDate || toolBusy.DownloadOffered {
		t.Errorf("the patch chain reads %v offered %v during a tool check", toolBusy.Update, toolBusy.DownloadOffered)
	}
	if toolBusy.Status.Busy != app.Idle || toolBusy.UpdateInFlight != app.Idle {
		t.Errorf("the patch slot reads busy %v in flight %v during a tool check", toolBusy.Status.Busy, toolBusy.UpdateInFlight)
	}
	if !toolBusy.Busy {
		t.Error("the gate reports idle while the tool check runs")
	}

	patchBusy := app.Derive(app.Facts{ToolVersion: "0.1.0", Busy: app.BusyCheckUpdate})
	if patchBusy.Tool.State != app.ToolUnchecked {
		t.Errorf("the tool check reads %v during a patch check", patchBusy.Tool.State)
	}
}
