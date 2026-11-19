package main

import (
	"log"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/deploy"
	"opensteamtool-gui-mygo/internal/source"
)

type uninstallRun struct {
	stop    bool
	restart bool
}

func (s *session) uninstall() bool {
	return s.beginUninstall(uninstallRun{})
}

func (s *session) exitSteamAndUninstall() bool {
	return s.request(app.ConfirmExitRemove, uninstallRun{stop: true})
}

func (s *session) uninstallAndRestart() bool {
	return s.request(app.ConfirmRemoveAndRestart, uninstallRun{restart: true})
}

func (s *session) request(pending app.Confirmation, run uninstallRun) bool {
	if !s.idle() {
		return false
	}
	if s.factsWithoutGate().SteamRunning {
		s.setPending(pending)
		return true
	}
	run.stop = false
	return s.beginUninstall(run)
}

func (s *session) idle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gate.Busy() == app.Idle
}

func (s *session) setPending(pending app.Confirmation) {
	s.mu.Lock()
	s.facts.Pending = pending
	s.mu.Unlock()
	s.draw()
}

func (s *session) declinePending() {
	s.setPending(app.NoConfirmation)
}

func (s *session) confirmPending() {
	s.mu.Lock()
	pending := s.facts.Pending
	s.facts.Pending = app.NoConfirmation
	s.mu.Unlock()
	switch pending {
	case app.ConfirmExitRemove:
		s.beginUninstall(uninstallRun{stop: true})
	case app.ConfirmRemoveAndRestart:
		s.beginUninstall(uninstallRun{stop: true, restart: true})
	}
}

func (s *session) beginUninstall(run uninstallRun) bool {
	if !s.take(app.BusyUninstall) {
		return false
	}
	go s.runUninstall(run)
	return true
}

func (s *session) runUninstall(run uninstallRun) {
	facts := s.factsWithoutGate()
	root := facts.SteamPath
	if !facts.SteamPathValid {
		s.finishUninstall(app.PathUnset)
		return
	}
	if run.stop {
		s.advance(app.BusyStopSteam)
		if err := s.client.Stop(root); err != nil {
			log.Printf("close Steam for the removal: %v", err)
			s.finishUninstall(app.CloseSteamFailed)
			return
		}
		s.advance(app.BusyUninstall)
	}
	src := source.OrDefault(s.config().Source)
	if err := deploy.Uninstall(s.host, deploy.Removal{Root: root, Layout: src.Layout, ToolDir: s.dir, Source: src}); err != nil {
		log.Printf("remove the patch: %v", err)
		s.finishUninstall(app.RemovalFailed)
		return
	}
	if run.restart {
		s.advance(app.BusyStartSteam)
		if err := s.client.Start(root); err != nil {
			log.Printf("start Steam after the removal: %v", err)
			s.finishUninstall(startResult(err, app.NoResult))
			return
		}
	}
	s.finishUninstall(app.PatchRemoved)
}

func (s *session) finishUninstall(result app.Result) {
	s.onUI(func() {
		next := withConfig(s.factsWithoutGate(), s.config(), s.host, s.dir)
		s.mu.Lock()
		s.facts = next
		s.mu.Unlock()
		s.done(result)
	})
}
