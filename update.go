package main

import (
	"errors"
	"log"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
)

func (s *session) checkUpdate() bool {
	if !s.take(app.BusyCheckUpdate) {
		return false
	}
	go s.runCheck()
	return true
}

func (s *session) runCheck() {
	src := source.OrDefault(s.config().Source)
	release, err := s.latest(src)
	if err != nil {
		s.finishCheck("", patch.Asset{}, err)
		return
	}
	asset, err := patch.Pick(release.Assets)
	if err != nil {
		s.finishCheck(release.Version, patch.Asset{}, err)
		return
	}
	s.finishCheck(release.Version, asset, nil)
}

func (s *session) latest(src source.Source) (patch.Release, error) {
	body, err := s.host.Get(patch.LatestURL(src.Repository))
	if err != nil {
		return patch.Release{}, err
	}
	return patch.Parse(body)
}

func (s *session) dropOnline() {
	s.facts.UpdateChecked = false
	s.facts.UpdateFailed = false
	s.facts.OnlineVersion = ""
	s.online = online{}
}

func checkFailure(err error) app.Result {
	switch {
	case errors.Is(err, patch.ErrUnparsed):
		return app.ParseFailed
	case errors.Is(err, patch.ErrArchivesOnly):
		return app.ArchivesOnly
	case errors.Is(err, patch.ErrNoAsset):
		return app.NoAsset
	}
	return app.NetworkFailed
}

func (s *session) finishCheck(version string, asset patch.Asset, err error) {
	failure := app.NoResult
	if err != nil {
		log.Printf("check the patch release: %v", err)
		failure = checkFailure(err)
	}
	s.onUI(func() {
		s.mu.Lock()
		s.facts.UpdateChecked = err == nil
		s.facts.UpdateFailed = err != nil
		s.facts.OnlineVersion = version
		s.online = online{asset: asset, version: version}
		s.mu.Unlock()
		s.done(failure)
	})
}

func (s *session) downloadPatch() bool {
	if !app.Derive(s.snapshot()).DownloadOffered {
		return false
	}
	if !s.take(app.BusyDownloadPatch) {
		return false
	}
	go s.runDownload()
	return true
}

func extractFailure(err error) app.Result {
	if errors.Is(err, patch.ErrMissingMember) {
		return app.MissingMember
	}
	return app.LocalIOFailed
}

func (s *session) runDownload() {
	s.mu.Lock()
	chosen := s.online
	s.mu.Unlock()
	cfg := s.config()
	src := source.OrDefault(cfg.Source)
	entry := source.Effective(src, cfg.Entry)
	archive, err := s.host.Get(chosen.asset.URL)
	if err != nil {
		log.Printf("fetch the patch package: %v", err)
		s.onUI(func() { s.done(app.NetworkFailed) })
		return
	}
	files, err := patch.Extract(archive, src.Layout, entry)
	if err != nil {
		log.Printf("read the patch package: %v", err)
		s.onUI(func() { s.done(extractFailure(err)) })
		return
	}
	if err := patch.Install(patch.Dir(s.dir, src), files, chosen.version); err != nil {
		log.Printf("fill the patch cache: %v", err)
		s.onUI(func() { s.done(app.WriteFailed) })
		return
	}
	s.onUI(func() {
		next := withConfig(s.facts, s.cfg, s.host, s.dir)
		s.mu.Lock()
		s.facts = next
		s.mu.Unlock()
		s.done(app.Downloaded)
	})
}
