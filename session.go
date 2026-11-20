package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/compat"
	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

type session struct {
	host    rt.Runtime
	client  *steam.Client
	sleep   func(time.Duration)
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

	compat      *compat.Round
	compatStamp string

	frame       func()
	windowWidth func() float32
	retitle     func(string)
	retray      func()

	visible func() bool
	hide    func()
	show    func()
	notify  func(string)
	quit    func()
	now     func() time.Time

	systemDark func() (bool, bool)
	applyDark  func(bool)
	watchTheme func(func()) func()
	stopTheme  func()

	tray      bool
	quitting  bool
	ballooned bool
	painted   time.Time
	steps     atomic.Int32
}

func newSession(host rt.Runtime, sleep func(time.Duration), locale string, cfg config.Config, readable bool, dir string) *session {
	s := &session{
		host:    host,
		client:  steam.New(host, sleep),
		sleep:   sleep,
		cfg:     cfg,
		dir:     dir,
		frame:   func() {},
		onUI:    func(fn func()) { fn() },
		visible: func() bool { return true },
		hide:    func() {},
		show:    func() {},
		notify:  func(string) {},
		quit:    func() {},
		retray:  func() {},
		now:     time.Now,
		compat:  compat.NewRound(),

		systemDark: systemAppearance,
		applyDark:  applyAppearance,
		watchTheme: mygo.Theme.OnUpdated,
		stopTheme:  func() {},
	}
	s.facts = factsFor(cfg.Language, locale, cfg, host, s.dir)
	s.actions = view.Actions{
		SetLanguage:         s.setLanguage,
		SetTheme:            s.setTheme,
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
		Prewarm:             func() { s.prewarm(true) },
		CheckToolUpdate:     func() { s.checkToolUpdate() },
		OpenProject:         func() { s.openURL(projectURL) },
		OpenDownload:        func() { s.openURL(releasesURL) },
		WizardNext:          s.wizardNext,
		WizardBack:          s.wizardBack,
		WizardRerun:         s.rerunWizard,
		WizardLanguage:      s.setWizardLanguage,
		WizardTheme:         s.setWizardTheme,
		WizardDownload:      func() { s.startWizardDownload() },
		WizardSkip:          s.finishWizard,
		WizardFinish:        s.finishWizard,
	}
	if !readable {
		s.openWizard()
	}
	return s
}

func (s *session) view(c *ui.Context) {
	facts := s.snapshot()
	if facts.Wizard.Open {
		s.syncWizard(facts)
	} else if !s.uiState.SettingsOpen {
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
	s.uiState.Theme = int(facts.Theme)
}

func (s *session) snapshot() app.Facts {
	facts := s.factsWithoutGate()
	facts.Busy = s.gate.Busy()
	if s.windowWidth != nil {
		facts.WindowWidth = s.windowWidth()
	}
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
	s.painted = s.now()
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
