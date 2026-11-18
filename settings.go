package main

import (
	"log"
	"strings"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

func factsFor(language app.Language, locale, steamPath string, host rt.Runtime) app.Facts {
	return withPath(app.Facts{Language: language, Locale: locale}, steamPath, host)
}

func withPath(facts app.Facts, steamPath string, host rt.Runtime) app.Facts {
	facts.SteamPath = steamPath
	facts.SteamPathValid = steamPath != "" && host.IsDir(steamPath)
	facts.Deployment = app.NotApplied
	if running, err := steam.Running(host, steamPath); err == nil {
		facts.SteamRunning = running
	}
	if facts.SteamPathValid {
		facts.Deployment = app.DeploymentOf(app.OpenSteamToolLayout(), "", steamPath, host)
	}
	return facts
}

func (s *session) commitSteamPath(raw string) bool {
	path := strings.TrimSpace(raw)
	if path != "" && (!s.host.IsDir(path) || !s.host.IsReadableDir(path)) {
		return false
	}
	current := s.factsWithoutGate()
	if path == current.SteamPath {
		return true
	}
	facts := withPath(current, path, s.host)

	s.mu.Lock()
	s.facts = facts
	s.cfg.SteamPath = path
	cfg := s.cfg
	s.mu.Unlock()

	s.uiState.PathDraft = path
	s.save(cfg)
	s.draw()
	return true
}

func (s *session) setLanguage(language app.Language) {
	s.mu.Lock()
	s.facts.Language = language
	s.cfg.Language = language
	cfg, locale := s.cfg, s.facts.Locale
	s.mu.Unlock()

	s.save(cfg)
	if s.retitle != nil {
		s.retitle(view.Title(language.Resolve(locale)))
	}
	s.draw()
}

func (s *session) browseSteamPath() {
	start := s.factsWithoutGate().SteamPath
	go s.pickSteamPath(start)
}

func (s *session) pickSteamPath(start string) {
	path, err := s.host.PickDirectory(view.SteamDirectory(s.resolvedLanguage()), start)
	if err != nil {
		log.Printf("pick the Steam directory: %v", err)
		return
	}
	if path == "" {
		return
	}
	s.onUI(func() { s.commitSteamPath(path) })
}

func (s *session) resolvedLanguage() app.Lang {
	facts := s.factsWithoutGate()
	return facts.Language.Resolve(facts.Locale)
}

func (s *session) save(cfg config.Config) {
	if s.persist == nil {
		return
	}
	if err := s.persist(cfg); err != nil {
		log.Printf("save config: %v", err)
		return
	}
	s.mu.Lock()
	s.facts.NeedsSetup = false
	s.mu.Unlock()
}

func (s *session) watch(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if s.refresh() {
			s.draw()
		}
	}
}
