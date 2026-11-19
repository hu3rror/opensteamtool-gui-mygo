package main

import (
	"errors"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

type session struct {
	host    rt.Runtime
	client  *steam.Client
	persist func(config.Config) error
	onUI    func(func())
	dir     string

	uiState view.State
	actions view.Actions

	mu     sync.Mutex
	facts  app.Facts
	cfg    config.Config
	gate   app.Gate
	online online

	frame   func()
	retitle func(string)
	hide    func()
}

func newSession(host rt.Runtime, sleep func(time.Duration), locale string, cfg config.Config, readable bool, dir string) *session {
	s := &session{
		host:   host,
		client: steam.New(host, sleep),
		cfg:    cfg,
		dir:    dir,
		frame:  func() {},
		onUI:   func(fn func()) { fn() },
	}
	s.facts = factsFor(cfg.Language, locale, cfg, host, s.dir)
	s.facts.NeedsSetup = !readable
	s.actions = view.Actions{
		SetLanguage:         s.setLanguage,
		CommitPath:          s.commitSteamPath,
		Browse:              s.browseSteamPath,
		ApplySource:         s.applySource,
		SetEntry:            s.setEntry,
		CheckUpdate:         func() { s.checkUpdate() },
		Download:            func() { s.downloadPatch() },
		Apply:               func() { s.applyPatch() },
		RemoveConflict:      func() { s.removeConflictAndDeploy() },
		Uninstall:           func() { s.uninstall() },
		ExitAndUninstall:    func() { s.exitSteamAndUninstall() },
		UninstallAndRestart: func() { s.uninstallAndRestart() },
		Confirm:             s.confirmPending,
		Cancel:              s.declinePending,
	}
	return s
}

func (s *session) view(c *ui.Context) {
	facts := s.snapshot()
	if !s.uiState.SettingsOpen {
		s.syncSettings(facts)
		s.syncSource()
	}
	view.Main(c, app.Derive(facts), &s.uiState, s.actions)
}

func (s *session) syncSettings(facts app.Facts) {
	s.uiState.PathDraft = facts.SteamPath
	s.uiState.PathError = false
	s.uiState.PathFocused = false
	s.uiState.Language = int(facts.Language)
}

func (s *session) snapshot() app.Facts {
	facts := s.factsWithoutGate()
	facts.Busy = s.gate.Busy()
	return facts
}

func (s *session) factsWithoutGate() app.Facts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.facts
}

func (s *session) refresh() bool {
	facts := s.factsWithoutGate()
	running, err := steam.Running(s.host, facts.SteamPath)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.facts.SteamRunning != running
	s.facts.SteamRunning = running
	return changed
}

func (s *session) restartSteam() bool {
	if !s.take(app.BusyStopSteam) {
		return false
	}
	root := s.factsWithoutGate().SteamPath
	result := app.NoResult
	if err := s.client.Stop(root); err != nil {
		result = app.CloseSteamFailed
	} else {
		s.advance(app.BusyStartSteam)
		result = startResult(s.client.Start(root), app.SteamRestarted)
	}
	s.done(result)
	return true
}

func (s *session) take(kind app.Busy) bool {
	s.mu.Lock()
	taken := s.gate.Try(kind)
	s.mu.Unlock()
	if taken {
		s.draw()
	}
	return taken
}

func (s *session) advance(kind app.Busy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gate.Advance(kind)
}

func (s *session) done(result app.Result) {
	running, err := steam.Running(s.host, s.factsWithoutGate().SteamPath)
	s.mu.Lock()
	s.gate.Release()
	if result != app.NoResult {
		s.facts.Result = result
	}
	if err == nil {
		s.facts.SteamRunning = running
	}
	s.mu.Unlock()
	s.draw()
}

func (s *session) draw() {
	if s.frame != nil {
		s.frame()
	}
}

func startResult(err error, started app.Result) app.Result {
	switch {
	case err == nil:
		return started
	case errors.Is(err, steam.ErrMissing):
		return app.MissingSteamProgram
	default:
		return app.StartSteamFailed
	}
}
