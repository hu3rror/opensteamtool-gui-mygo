package main

import (
	"log"
	"path/filepath"
	"sync/atomic"
	"time"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/deploy"
	"opensteamtool-gui-mygo/internal/source"
	"opensteamtool-gui-mygo/internal/view"
)

const (
	hiddenRepaint = time.Minute
	quitGrace     = 50 * time.Millisecond
)

func (s *session) closeWindow() bool {
	if s.quitting || !s.tray {
		return true
	}
	s.hideToTray()
	return false
}

func (s *session) toggle() {
	if s.visible() {
		s.hideToTray()
		return
	}
	s.wake()
}

func (s *session) hideToTray() {
	if !s.visible() {
		return
	}
	s.hide()
	if s.ballooned {
		return
	}
	s.ballooned = true
	s.notify(view.TrayRunning(s.resolvedLanguage()))
}

func (s *session) wake() {
	if s.visible() {
		return
	}
	s.show()
	s.draw()
}

func (s *session) requestExit() {
	s.wake()
	s.setPending(app.ConfirmExit)
}

func (s *session) exit() {
	s.quitting = true
	for i := 0; i < int(quitGrace/time.Millisecond) && s.steps.Load() > 0; i++ {
		s.sleep(time.Millisecond)
	}
	s.stopTheme()
	s.quit()
}

func (s *session) poll() {
	s.startCompat()
	was := s.factsWithoutGate().SteamRunning
	if !s.refresh() {
		return
	}
	s.followSteam(was)
	s.paint()
}

func (s *session) followSteam(wasRunning bool) {
	running := s.factsWithoutGate().SteamRunning
	switch {
	case running && !wasRunning:
		s.hideToTray()
	case wasRunning && !running:
		s.wake()
	}
}

func (s *session) paint() {
	if s.visible() || s.now().Sub(s.painted) >= hiddenRepaint {
		s.draw()
	}
}

func (s *session) luaDir() (string, bool) {
	facts := s.factsWithoutGate()
	if !facts.SteamPathValid {
		return "", false
	}
	lua := source.OrDefault(s.config().Source).Layout.Manifest().Lua
	return filepath.Join(facts.SteamPath, filepath.FromSlash(lua)), true
}

func (s *session) openLuaFolder() {
	dir, ok := s.luaDir()
	if !ok {
		return
	}
	err := s.host.MkdirAll(dir)
	if err == nil {
		err = s.host.OpenPath(dir)
	}
	if err == nil {
		return
	}
	log.Printf("open the Lua directory: %v", err)
	s.mu.Lock()
	s.facts.Result = app.LocalIOFailed
	s.mu.Unlock()
	s.draw()
}

type fileSteps struct {
	deploy.Files
	steps *atomic.Int32
}

func (f fileSteps) WriteFile(path string, data []byte) error {
	f.steps.Add(1)
	defer f.steps.Add(-1)
	return f.Files.WriteFile(path, data)
}

func (f fileSteps) Rename(from string, to string) error {
	f.steps.Add(1)
	defer f.steps.Add(-1)
	return f.Files.Rename(from, to)
}

func (f fileSteps) Remove(path string) error {
	f.steps.Add(1)
	defer f.steps.Add(-1)
	return f.Files.Remove(path)
}

func (s *session) files() deploy.Files {
	return fileSteps{Files: s.host, steps: &s.steps}
}
