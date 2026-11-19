package main

import (
	"errors"
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

func toolRelease() map[string][]byte {
	return map[string][]byte{
		toolReleaseURL(): []byte(`{"tag_name": "v0.2.0", "assets": [{"name": "opensteamtool-mygo-release.zip", "browser_download_url": "https://example.test/tool-release.zip"}]}`),
	}
}

func TestTheSelfUpdateCheckReadsTheReleaseAndDownloadsNothing(t *testing.T) {
	host := &rt.Fake{Responses: toolRelease()}
	s := newTestSession(t, host, "")
	s.setToolVersion("0.1.0")

	runCommand(t, s, s.checkToolUpdate)

	facts := s.snapshot()
	if !facts.ToolChecked || facts.ToolFailed || facts.ToolOnline != "v0.2.0" {
		t.Fatalf("facts = %+v, want the release read once", facts)
	}
	if facts.Busy != app.Idle {
		t.Fatalf("busy = %v, the gate stayed taken", facts.Busy)
	}
	if facts.UpdateChecked || facts.UpdateFailed || facts.OnlineVersion != "" {
		t.Errorf("facts = %+v, the patch chain took the tool check", facts)
	}
	if len(host.Fetched) != 1 || len(host.Writes) != 0 {
		t.Errorf("a tool check fetched %v and wrote %v, want one read and no writes", host.Fetched, host.Writes)
	}
}

func TestAFailedSelfUpdateCheckIsRecorded(t *testing.T) {
	host := &rt.Fake{FetchError: errors.New("offline")}
	s := newTestSession(t, host, "")

	runCommand(t, s, s.checkToolUpdate)

	facts := s.snapshot()
	if facts.ToolChecked || !facts.ToolFailed || facts.ToolOnline != "" {
		t.Fatalf("facts = %+v, want the failure kept apart", facts)
	}
	if facts.Busy != app.Idle {
		t.Fatalf("busy = %v, the gate stayed taken", facts.Busy)
	}
	if facts.UpdateFailed || facts.UpdateChecked {
		t.Errorf("facts = %+v, the patch chain took the tool failure", facts)
	}
}

func TestNoSelfUpdateRequestArrivesWithoutAClick(t *testing.T) {
	host := &rt.Fake{Responses: toolRelease()}
	s := newTestSession(t, host, "")
	shown(t, s)

	if len(host.Fetched) != 0 {
		t.Fatalf("fetched %v without a click", host.Fetched)
	}
	if facts := s.snapshot(); facts.ToolChecked || facts.ToolFailed {
		t.Errorf("facts = %+v, want no check before one is asked for", facts)
	}
}

func TestTheProjectLinksOpenInTheBrowser(t *testing.T) {
	host := &rt.Fake{}
	s := newTestSession(t, host, "")

	s.actions.OpenProject()
	s.actions.OpenDownload()

	if !slices.Equal(host.OpenedURLs, []string{projectURL, releasesURL}) {
		t.Errorf("opened %v, want the project page and the download page", host.OpenedURLs)
	}
}
