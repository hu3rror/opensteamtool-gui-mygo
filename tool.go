package main

import (
	"log"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/patch"
)

const (
	projectRepo = "hu3rror/opensteamtool-gui-mygo"
	projectURL  = "https://github.com/" + projectRepo
	releasesURL = projectURL + "/releases"
)

func toolReleaseURL() string {
	return patch.LatestURL(projectRepo)
}

func (s *session) setToolVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts.ToolVersion = version
}

func (s *session) checkToolUpdate() bool {
	if !s.take(app.BusyCheckTool) {
		return false
	}
	go s.runToolCheck()
	return true
}

func (s *session) runToolCheck() {
	version, err := s.latestTool()
	if err != nil {
		log.Printf("check the tool release: %v", err)
	}
	s.onUI(func() {
		s.mu.Lock()
		s.facts.ToolChecked = err == nil
		s.facts.ToolFailed = err != nil
		s.facts.ToolOnline = version
		s.mu.Unlock()
		s.done(app.NoResult)
	})
}

func (s *session) latestTool() (string, error) {
	body, err := s.host.Get(toolReleaseURL())
	if err != nil {
		return "", err
	}
	release, err := patch.Parse(body)
	if err != nil {
		return "", err
	}
	return release.Version, nil
}

func (s *session) openURL(url string) {
	if err := s.host.OpenURL(url); err != nil {
		log.Printf("open %s: %v", url, err)
	}
}
