package main

import (
	"slices"
	"testing"

	"opensteamtool-gui-mygo/internal/rt"
)

func TestTheProjectLinksOpenInTheBrowser(t *testing.T) {
	host := &rt.Fake{}
	s := newTestSession(t, host, "")

	s.actions.OpenProject()
	s.actions.OpenDownload()

	if !slices.Equal(host.OpenedURLs, []string{projectURL, releasesURL}) {
		t.Errorf("opened %v, want the project page and the download page", host.OpenedURLs)
	}
}
