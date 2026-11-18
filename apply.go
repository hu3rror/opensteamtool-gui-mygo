package main

import (
	"errors"
	"log"
	"path/filepath"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/deploy"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
	"opensteamtool-gui-mygo/internal/steam"
)

func (s *session) applyPatch() bool {
	return s.beginApply("")
}

func (s *session) removeConflictAndDeploy() bool {
	return s.beginApply(s.factsWithoutGate().Conflict)
}

func (s *session) beginApply(remove string) bool {
	if !s.take(app.BusyApplyPatch) {
		return false
	}
	go s.runApply(remove)
	return true
}

func (s *session) runApply(remove string) {
	facts := s.factsWithoutGate()
	cfg := s.config()
	src := source.OrDefault(cfg.Source)
	entry := source.Effective(src, cfg.Entry)
	root := facts.SteamPath
	switch {
	case !facts.SteamPathValid:
		s.failApply(app.PathUnset, "")
		return
	case !facts.PatchDownloaded || !facts.PatchComplete:
		s.failApply(app.MissingMember, "")
		return
	case !s.host.IsFile(filepath.Join(root, steam.Exe)):
		s.failApply(app.MissingSteamProgram, "")
		return
	}
	files, err := patch.Load(patch.Dir(s.dir, src), src.Layout, entry)
	if err != nil {
		log.Printf("read the patch cache: %v", err)
		s.failApply(app.MissingMember, "")
		return
	}
	if err := s.client.Stop(root); err != nil {
		log.Printf("close Steam for the patch: %v", err)
		s.failApply(app.CloseSteamFailed, "")
		return
	}
	plan := deploy.Plan{Root: root, Layout: src.Layout, Entry: entry, Files: files, Remove: remove}
	err = deploy.Write(s.host, plan)
	if err != nil {
		log.Printf("deploy the patch: %v", err)
		var conflict deploy.Conflict
		if !errors.As(err, &conflict) {
			s.failApply(app.DeploymentFailed, "")
			return
		}
		s.failApply(app.DeploymentFailed, conflict.Name)
		return
	}
	build := patch.ClientBuild(root)
	s.advance(app.BusyStartSteam)
	if err := s.client.Start(root); err != nil {
		log.Printf("start Steam for the patch: %v", err)
		s.failApply(startResult(err, app.PatchApplied), "")
		return
	}
	s.recordApply(src, build)
}

func (s *session) recordApply(src source.Source, build string) {
	s.onUI(func() {
		s.mu.Lock()
		s.facts.PatchFailed = false
		s.facts.Conflict = ""
		version := s.facts.PatchVersion
		s.mu.Unlock()
		s.writeConfig(func(cfg *config.Config) {
			cfg.Snapshot = config.DeploymentSnapshot{Source: src.Key, Layout: src.Layout, ClientBuild: build, PatchVersion: version}
		})
		s.done(app.PatchApplied)
	})
}

func (s *session) failApply(result app.Result, conflict string) {
	s.onUI(func() {
		s.mu.Lock()
		s.facts.PatchFailed = true
		s.facts.Conflict = conflict
		s.mu.Unlock()
		s.done(result)
	})
}
