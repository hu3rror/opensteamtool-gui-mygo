package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestThePatchCheckLandsInOneOfFourConclusions(t *testing.T) {
	tests := []struct {
		name       string
		facts      app.Facts
		unchecked  bool
		moved      bool
		unrecorded bool
		want       app.Conclusion
		download   bool
	}{
		{
			name:      "nothing is said before a check",
			facts:     app.Facts{},
			unchecked: true,
			want:      app.NoConclusion,
		},
		{
			name:     "a local version without files still offers the download",
			facts:    app.Facts{PatchDownloaded: true, PatchVersion: "v1", OnlineVersion: "v1"},
			want:     app.NewPatch,
			download: true,
		},
		{
			name:  "the same version with every file is up to date",
			facts: app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1", OnlineVersion: "v1"},
			want:  app.UpToDate,
		},
		{
			name:  "the tag prefix and the case do not count",
			facts: app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1.2.0", OnlineVersion: "1.2.0"},
			want:  app.UpToDate,
		},
		{
			name:     "a newer version is a new patch",
			facts:    app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1", OnlineVersion: "v2"},
			want:     app.NewPatch,
			download: true,
		},
		{
			name:     "an older local version is still different",
			facts:    app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v2", OnlineVersion: "v1"},
			want:     app.NewPatch,
			download: true,
		},
		{
			name:     "a client that moved on warns and still offers the download",
			facts:    app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1", OnlineVersion: "v2"},
			moved:    true,
			want:     app.ClientUpdated,
			download: true,
		},
		{
			name:  "a client that moved on keeps an up to date patch quiet",
			facts: app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1", OnlineVersion: "v1"},
			moved: true,
			want:  app.ClientUpdated,
		},
		{
			name:     "a client that moved on without local files stays a new patch",
			facts:    app.Facts{OnlineVersion: "v1"},
			moved:    true,
			want:     app.NewPatch,
			download: true,
		},
		{
			name:       "a snapshot from another source or layout never warns",
			facts:      app.Facts{PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1", OnlineVersion: "v1"},
			unrecorded: true,
			want:       app.UpToDate,
		},
		{
			name:  "a failed check says so",
			facts: app.Facts{UpdateFailed: true},
			want:  app.CheckFailed,
		},
		{
			name:  "a failed check never reads as up to date",
			facts: app.Facts{UpdateFailed: true, PatchDownloaded: true, PatchComplete: true, PatchVersion: "v1"},
			want:  app.CheckFailed,
		},
		{
			name:  "a check without an online version says it failed",
			facts: app.Facts{},
			want:  app.CheckFailed,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts := test.facts
			facts.SteamPathValid = true
			facts.UpdateChecked = !test.unchecked
			facts.ClientBuild = "100"
			switch {
			case test.moved:
				facts.SnapshotBuild = "99"
			case test.unrecorded:
				facts.SnapshotBuild = ""
			default:
				facts.SnapshotBuild = "100"
			}
			view := app.Derive(facts)
			if view.Update != test.want {
				t.Errorf("the conclusion is %d, want %d", view.Update, test.want)
			}
			if view.DownloadOffered != test.download {
				t.Errorf("download offered = %v, want %v", view.DownloadOffered, test.download)
			}
		})
	}
}

func TestAnUpdateNeverTakesTheBusySlot(t *testing.T) {
	for _, kind := range []app.Busy{app.BusyCheckUpdate, app.BusyDownloadPatch} {
		view := app.Derive(app.Facts{Busy: kind})
		if view.Status.Busy != app.Idle {
			t.Errorf("busy %d takes the status slot", kind)
		}
		if view.UpdateInFlight != kind {
			t.Errorf("busy %d is read as in flight %d", kind, view.UpdateInFlight)
		}
	}
	view := app.Derive(app.Facts{Busy: app.BusyApplyPatch})
	if view.Status.Busy != app.BusyApplyPatch || view.UpdateInFlight != app.Idle {
		t.Errorf("apply busy = %v, update busy = %v", view.Status.Busy, view.UpdateInFlight)
	}
	if view := app.Derive(app.Facts{}); view.UpdateInFlight != app.Idle {
		t.Error("an idle snapshot reports an update in flight")
	}
}
