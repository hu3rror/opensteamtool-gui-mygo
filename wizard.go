package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/source"
)

func getenv(name string) string { return os.Getenv(name) }

var steamRegistry = registryString

func wizardDownloadShape(facts app.Facts) app.WizardDownloadState {
	if facts.PatchDownloaded && facts.PatchComplete {
		return app.WizardDownloaded
	}
	return app.WizardNotDownloaded
}

func detectSteamPath(host rt.Runtime, lookup func(string) string, registry func(string, string) string) string {
	for _, probe := range []struct{ key, value string }{
		{`Software\Valve\Steam`, "SteamPath"},
		{`HKLM\SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"},
		{`HKLM\SOFTWARE\Valve\Steam`, "InstallPath"},
	} {
		path := filepath.FromSlash(registry(probe.key, probe.value))
		if path != "" && host.IsFile(filepath.Join(path, "steam.exe")) {
			return path
		}
	}
	for _, base := range []string{lookup("ProgramFiles(x86)"), lookup("ProgramFiles")} {
		if base == "" {
			continue
		}
		path := filepath.Join(base, "Steam")
		if host.IsFile(filepath.Join(path, "steam.exe")) {
			return path
		}
	}
	return ""
}

func (s *session) wizardOpen() bool {
	return s.factsWithoutGate().Wizard.Open
}

func (s *session) openWizard() {
	cfg := s.config()
	facts := s.factsWithoutGate()
	path := facts.SteamPath
	if path == "" || !s.host.IsReadableDir(path) {
		path = detectSteamPath(s.host, getenv, s.registry)
	}
	s.mu.Lock()
	s.facts.Wizard = app.Wizard{
		Open:     true,
		Step:     app.WizardLanguage,
		Language: cfg.Language,
		Theme:    cfg.Theme,
		Download: wizardDownloadShape(s.facts),
	}
	s.mu.Unlock()
	s.uiState.PathDraft = path
	s.uiState.PathError = false
	s.uiState.PathNotFound = path == ""
	s.uiState.PathFocused = false
	s.syncSource()
	if path != "" && path != facts.SteamPath {
		s.commitSteamPath(path)
	}
	if s.frame != nil {
		s.draw()
	}
}

func (s *session) syncWizard(facts app.Facts) {
	s.uiState.WizardLanguage = int(facts.Wizard.Language)
	s.uiState.WizardTheme = int(facts.Wizard.Theme)
}

func (s *session) setWizardLanguage(language app.Language) {
	s.mu.Lock()
	s.facts.Wizard.Language = language
	s.facts.Language = language
	s.mu.Unlock()
	s.draw()
}

func (s *session) setWizardTheme(theme app.Theme) {
	s.mu.Lock()
	s.facts.Wizard.Theme = theme
	s.mu.Unlock()
	s.applyTheme()
	s.draw()
}

func (s *session) setWizardStep(step app.WizardStep) {
	s.mu.Lock()
	s.facts.Wizard.Step = step
	if step == app.WizardDownload {
		s.facts.Wizard.Download = wizardDownloadShape(s.facts)
		s.facts.Wizard.Failure = app.NoResult
	}
	s.mu.Unlock()
	s.draw()
}

func (s *session) wizardNext() {
	switch s.factsWithoutGate().Wizard.Step {
	case app.WizardLanguage:
		s.setWizardStep(app.WizardTheme)
	case app.WizardTheme:
		s.setWizardStep(app.WizardPath)
	case app.WizardPath:
		if !s.commitSteamPath(s.uiState.PathDraft) {
			s.uiState.PathError = true
			return
		}
		if !s.factsWithoutGate().SteamPathValid {
			return
		}
		s.setWizardStep(app.WizardDownload)
	}
}

func (s *session) wizardBack() {
	switch s.factsWithoutGate().Wizard.Step {
	case app.WizardTheme:
		s.setWizardStep(app.WizardLanguage)
	case app.WizardPath:
		s.setWizardStep(app.WizardTheme)
	case app.WizardDownload:
		s.setWizardStep(app.WizardPath)
	}
}

func (s *session) rerunWizard() {
	if s.wizardOpen() {
		return
	}
	s.openWizard()
}

func (s *session) finishWizard() {
	facts := s.factsWithoutGate()
	wizard := facts.Wizard
	if !wizard.Open {
		return
	}
	draft := s.uiState.Draft()
	resolved, err := draft.Resolve()
	path := strings.TrimSpace(s.uiState.PathDraft)
	themeChanged := s.config().Theme != wizard.Theme
	s.mu.Lock()
	s.facts.Wizard = app.Wizard{}
	s.mu.Unlock()
	s.writeConfig(func(cfg *config.Config) {
		cfg.Language = wizard.Language
		cfg.Theme = wizard.Theme
		if err == nil && source.OrDefault(resolved).Key != source.OrDefault(cfg.Source).Key {
			cfg.Source = source.OrDefault(resolved)
			cfg.Entry = source.Effective(cfg.Source, cfg.Entry)
		}
		if s.host.IsDir(path) && s.host.IsReadableDir(path) {
			cfg.SteamPath = path
		}
	})
	if themeChanged {
		s.applyTheme()
	}
	s.uiState.SetSource(source.OrDefault(s.config().Source))
	s.draw()
}

func (s *session) setWizardDownload(state app.WizardDownloadState, failure app.Result) {
	s.mu.Lock()
	s.facts.Wizard.Download = state
	s.facts.Wizard.Failure = failure
	if state == app.WizardDownloaded {
		wizard := s.facts.Wizard
		next := withConfig(s.facts, s.cfg, s.host, s.dir)
		next.Wizard = wizard
		next.Language = wizard.Language
		s.facts = next
	}
	s.mu.Unlock()
	s.draw()
}

func (s *session) startWizardDownload() bool {
	if !s.take(app.BusyDownloadPatch) {
		return false
	}
	s.setWizardDownload(app.WizardDownloading, app.NoResult)
	go s.runWizardDownload()
	return true
}

func (s *session) wizardDownloadDone(state app.WizardDownloadState, failure app.Result) {
	s.onUI(func() {
		s.mu.Lock()
		s.gate.Release()
		s.mu.Unlock()
		s.setWizardDownload(state, failure)
	})
}

func (s *session) runWizardDownload() {
	cfg := s.config()
	src := source.OrDefault(cfg.Source)
	fail := func(result app.Result) { s.wizardDownloadDone(app.WizardDownloadFailed, result) }
	release, err := s.latest(src)
	if err != nil {
		log.Printf("fetch the patch release: %v", err)
		fail(checkFailure(err))
		return
	}
	asset, err := patch.Pick(release.Assets)
	if err != nil {
		log.Printf("pick the patch package: %v", err)
		fail(checkFailure(err))
		return
	}
	if failure := s.fetchPackage(src, source.Effective(src, cfg.Entry), asset, release.Version); failure != app.NoResult {
		fail(failure)
		return
	}
	s.wizardDownloadDone(app.WizardDownloaded, app.NoResult)
}
