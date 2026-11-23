package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestTheViewCarriesTheAvailableVersionAndTheTwoPreferences(t *testing.T) {
	tests := []struct {
		name      string
		facts     app.Facts
		available string
		checks    bool
		downloads bool
	}{
		{
			name:   "before a check",
			facts:  app.Facts{AppVersion: "0.5.0", AutomaticChecks: true},
			checks: true,
		},
		{
			name:      "a version is available",
			facts:     app.Facts{AppVersion: "0.5.0", UpdateAvailable: "0.6.0", AutomaticChecks: true, AutomaticDownloads: true},
			available: "0.6.0",
			checks:    true,
			downloads: true,
		},
		{
			name:  "both preferences off",
			facts: app.Facts{AppVersion: "0.5.0"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := app.Derive(test.facts)
			if v.AppVersion != test.facts.AppVersion {
				t.Errorf("version = %q, want %q", v.AppVersion, test.facts.AppVersion)
			}
			if v.UpdateAvailable != test.available {
				t.Errorf("available = %q, want %q", v.UpdateAvailable, test.available)
			}
			if v.AutomaticChecks != test.checks || v.AutomaticDownloads != test.downloads {
				t.Errorf("preferences = checks %v downloads %v, want checks %v downloads %v",
					v.AutomaticChecks, v.AutomaticDownloads, test.checks, test.downloads)
			}
		})
	}
}
