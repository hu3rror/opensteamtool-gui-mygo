package main

import "log"

const (
	projectRepo = "hu3rror/opensteamtool-gui-mygo"
	projectURL  = "https://github.com/" + projectRepo
	releasesURL = projectURL + "/releases"
)

func (s *session) setAppVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts.AppVersion = version
}

func (s *session) setAutomaticChecks(on bool) {
	s.setChecks(on)
	s.refreshUpdate()
}

func (s *session) setAutomaticDownloads(on bool) {
	s.setDownloads(on)
	s.refreshUpdate()
}

func (s *session) refreshUpdate() {
	available := s.updateAvailable()
	checks := s.checksOn()
	downloads := s.downloadsOn()
	s.mu.Lock()
	s.facts.UpdateAvailable = available
	s.facts.AutomaticChecks = checks
	s.facts.AutomaticDownloads = downloads
	s.mu.Unlock()
	s.draw()
}

func (s *session) followUpdater() {
	s.stopUpdates = s.watchUpdates(s.refreshUpdate)
	s.refreshUpdate()
}

func (s *session) openURL(url string) {
	if err := s.host.OpenURL(url); err != nil {
		log.Printf("open %s: %v", url, err)
	}
}
