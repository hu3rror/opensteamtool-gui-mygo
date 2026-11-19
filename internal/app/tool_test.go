package app_test

import (
	"testing"

	"opensteamtool-gui-mygo/internal/app"
)

func TestTheToolVersionComparisonCountsSegments(t *testing.T) {
	tests := []struct {
		online string
		local  string
		newer  bool
	}{
		{online: "0.2.0", local: "0.1.0", newer: true},
		{online: "v0.2.0", local: "0.1.0", newer: true},
		{online: "0.1.0", local: "0.1.0"},
		{online: "0.1", local: "0.1.0"},
		{online: "0.1.1", local: "0.1", newer: true},
		{online: "0.1.0.0", local: "0.1"},
		{online: "1.0", local: "0.1.0", newer: true},
		{online: "0.10", local: "0.9", newer: true},
		{online: "0.9", local: "0.10"},
		{online: "nightly", local: "0.1"},
		{online: "nightly2", local: "0.1"},
		{online: "0.1.0-beta", local: "0.1.0"},
		{online: "", local: "0.0.0"},
	}
	for _, test := range tests {
		if got := app.ToolNewer(test.online, test.local); got != test.newer {
			t.Errorf("ToolNewer(%q, %q) = %v, want %v", test.online, test.local, got, test.newer)
		}
	}
}
