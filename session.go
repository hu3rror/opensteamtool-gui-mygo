package main

import (
	"errors"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

type session struct {
	steamPath string
	host      rt.Runtime
	client    *steam.Client

	mu    sync.Mutex
	facts app.Facts
	gate  app.Gate

	frame func()
}

func newSession(lang app.Lang, steamPath string, host rt.Runtime, sleep func(time.Duration)) *session {
	return &session{
		steamPath: steamPath,
		host:      host,
		client:    steam.New(host, sleep),
		facts:     factsFor(lang, steamPath, host),
	}
}

func (s *session) view(c *ui.Context) {
	view.Main(c, app.Derive(s.snapshot()))
}

func (s *session) snapshot() app.Facts {
	s.mu.Lock()
	defer s.mu.Unlock()
	facts := s.facts
	facts.Busy = s.gate.Busy()
	return facts
}

func (s *session) refresh() bool {
	running, err := steam.Running(s.host, s.steamPath)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := s.facts.SteamRunning != running
	s.facts.SteamRunning = running
	return changed
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

func (s *session) restartSteam() bool {
	if !s.take(app.BusyStopSteam) {
		return false
	}
	result := app.NoResult
	if err := s.client.Stop(s.steamPath); err != nil {
		result = app.CloseSteamFailed
	} else {
		s.advance(app.BusyStartSteam)
		result = startResult(s.client.Start(s.steamPath), app.SteamRestarted)
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
	running, err := steam.Running(s.host, s.steamPath)
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
