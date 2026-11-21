package main

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

type online struct {
	asset   patch.Asset
	version string
}

func factsFor(language app.Language, locale string, cfg config.Config, host rt.Runtime, dir string) app.Facts {
	return withConfig(app.Facts{Language: language, Locale: locale}, cfg, host, dir)
}

func luaFolder(cfg config.Config) (string, string) {
	if cfg.SteamPath == "" {
		return "", ""
	}
	lua := source.OrDefault(cfg.Source).Layout.Manifest().Lua
	return lua, filepath.Join(cfg.SteamPath, filepath.FromSlash(lua))
}

func withConfig(facts app.Facts, cfg config.Config, host rt.Runtime, dir string) app.Facts {
	facts.Language = cfg.Language
	facts.Theme = cfg.Theme
	facts.SteamPath = cfg.SteamPath
	facts.SteamPathValid = cfg.SteamPath != "" && host.IsDir(cfg.SteamPath)
	facts.LuaRelative, facts.LuaDir = "", ""
	if facts.SteamPathValid {
		facts.LuaRelative, facts.LuaDir = luaFolder(cfg)
	}
	facts.LuaManifest = facts.LuaDir != "" && host.IsFile(filepath.Join(facts.LuaDir, "manifest.lua"))
	facts.Deployment = app.NotApplied
	if running, err := steam.Running(host, cfg.SteamPath); err == nil {
		facts.SteamRunning = running
	}
	if facts.SteamPathValid {
		facts.Deployment = deploymentOf(cfg, host)
	}
	src := source.OrDefault(cfg.Source)
	state := patch.Inspect(patch.Dir(dir, src), src.Layout, source.Effective(src, cfg.Entry))
	facts.PatchDownloaded = state.Present
	facts.PatchComplete = state.Complete
	facts.PatchVersion = state.Version
	facts.SnapshotBuild = ""
	if cfg.Snapshot.Source == src.Key && cfg.Snapshot.Layout == src.Layout {
		facts.SnapshotBuild = cfg.Snapshot.ClientBuild
	}
	facts.ClientBuild = patch.ClientBuild(cfg.SteamPath)
	facts.PatchFailed = false
	facts.Conflict = ""
	return facts
}

func deploymentOf(cfg config.Config, files app.Files) app.Deployment {
	src := source.OrDefault(cfg.Source)
	manifest := src.Layout.Manifest()
	entry, _ := src.Layout.EntryFile(source.Effective(src, cfg.Entry))
	return app.DeploymentOf(app.Layout{
		Payload:         manifest.Payload.Name,
		SecondaryLoader: manifest.Loader.Name,
		DefaultEntry:    entry,
	}, "", cfg.SteamPath, files)
}

func (s *session) commitSteamPath(raw string) bool {
	path := strings.TrimSpace(raw)
	if path != "" && (!s.host.IsDir(path) || !s.host.IsReadableDir(path)) {
		return false
	}
	s.uiState.PathNotFound = false
	if path == s.config().SteamPath {
		return true
	}
	s.uiState.PathDraft = path
	s.writeConfig(func(cfg *config.Config) { cfg.SteamPath = path })
	s.startCompat()
	return true
}

func (s *session) detectSteam() {
	path := detectSteamPath(s.host, getenv, s.registry)
	if path == "" {
		s.uiState.PathNotFound = true
		s.draw()
		return
	}
	s.uiState.PathError = !s.commitSteamPath(path)
	s.draw()
}

func (s *session) setLanguage(language app.Language) {
	s.writeConfig(func(cfg *config.Config) { cfg.Language = language })
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
}

func (s *session) watch(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.poll()
	}
}

func (s *session) config() config.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *session) writeConfig(change func(*config.Config)) {
	s.mu.Lock()
	cfg := s.cfg
	previous := source.OrDefault(cfg.Source).Key
	change(&cfg)
	moved := source.OrDefault(cfg.Source).Key != previous
	s.cfg = cfg
	s.mu.Unlock()
	next := withConfig(s.facts, cfg, s.host, s.dir)
	s.mu.Lock()
	s.facts = next
	if moved {
		s.dropOnline()
	}
	s.mu.Unlock()
	s.save(cfg)
	s.draw()
	s.retray()
}

func (s *session) applySource(draft source.Draft) (source.Source, bool) {
	resolved, err := draft.Resolve()
	if err != nil {
		return source.Source{}, false
	}
	src := source.OrDefault(resolved)
	if src.Key == source.OrDefault(s.config().Source).Key {
		return src, true
	}
	s.writeConfig(func(cfg *config.Config) {
		cfg.Source = src
		cfg.Entry = source.Effective(src, cfg.Entry)
	})
	return src, true
}

func (s *session) setEntry(entry source.Entry) {
	cfg := s.config()
	if next := source.Effective(source.OrDefault(cfg.Source), entry); next != cfg.Entry {
		s.writeConfig(func(cfg *config.Config) { cfg.Entry = next })
	}
}

func (s *session) syncSource() {
	cfg := s.config()
	src := source.OrDefault(cfg.Source)
	s.uiState.SetSource(src)
	s.uiState.Entry = source.Effective(src, cfg.Entry)
}
