package main

import (
	"log"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/compat"
	"opensteamtool-gui-mygo/internal/patch"
)

func (s *session) startCompat() {
	facts := s.factsWithoutGate()
	stamp := facts.SteamPath + "\x00" + patch.ClientBuild(facts.SteamPath)
	s.mu.Lock()
	if s.compatStamp == stamp {
		s.mu.Unlock()
		return
	}
	s.compatStamp = stamp
	job := s.compat.Start(facts.SteamPath)
	s.mu.Unlock()
	go s.runCompat(job)
	s.syncCompat()
}

func (s *session) runCompat(job compat.Job) {
	report := compat.Probes(s.host, job.Path, s.dir, s.now)
	s.onUI(func() {
		s.mu.Lock()
		accepted := s.compat.Accept(job, report)
		if accepted && report.Err != nil {
			s.facts.Result = compat.Category(report.Err)
		}
		s.mu.Unlock()
		if !accepted {
			return
		}
		s.syncCompat()
		s.prewarm(false)
	})
}

func (s *session) syncCompat() {
	s.mu.Lock()
	snapshot := s.compat.Snapshot()
	s.facts.Health = snapshot.Health
	s.facts.Components = snapshot.Components
	s.facts.Prewarming = snapshot.Prewarming
	s.facts.PrecacheReady = snapshot.Ready
	s.facts.PrecacheFailure = snapshot.Failure
	if snapshot.Failure != app.NoResult {
		s.facts.Result = snapshot.Failure
	}
	s.mu.Unlock()
	s.draw()
}

func (s *session) prewarm(manual bool) {
	s.mu.Lock()
	job, probes, ok := compat.Job{}, []compat.Probe(nil), false
	if manual || s.compat.Snapshot().Health == app.HealthAdaptedNotCached {
		job, probes, ok = s.compat.BeginPrewarm(manual)
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	s.syncCompat()
	go s.runPrewarm(job, probes, manual)
}

func (s *session) runPrewarm(job compat.Job, probes []compat.Probe, manual bool) {
	done, err := compat.Prewarm(s.host, job.Path, s.dir, probes)
	if err != nil && manual {
		log.Printf("cache the core signatures: %v", err)
	}
	s.onUI(func() {
		s.mu.Lock()
		accepted := s.compat.Prewarmed(job, err, done)
		s.mu.Unlock()
		if !accepted {
			return
		}
		s.syncCompat()
	})
}
