package main

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/config"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

//go:embed resources/icon.png
var trayIcon []byte

const (
	minContentWidth  = 460
	minContentHeight = 620
)

func locale() string {
	langs := mygo.App.PreferredLanguages()
	if len(langs) == 0 {
		return ""
	}
	return langs[0]
}

func toolDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func main() {
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	dir := toolDir()
	cfg, readable, err := config.Load(dir, rt.OS{})
	if err != nil {
		log.Printf("read config: %v", err)
	}
	s := newSession(rt.OS{}, time.Sleep, locale(), cfg, readable, dir)
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }
	s.setToolVersion(mygo.App.Version())
	s.applyTheme()

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:          view.Title(),
			Width:          480,
			Height:         620,
			StateKey:       "main",
			Content:        ui.View(s.view),
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 48,
		})
		s.frame = win.Invalidate
		s.windowWidth = func() float32 { return float32(win.ContentBounds().Width) }
		s.onUI = win.Update
		s.visible = func() bool { return win.IsVisible() && !win.IsMinimized() }
		s.hide = win.Hide
		s.show = func() {
			win.Restore()
			win.Show()
			win.Focus()
		}
		s.quit = mygo.App.Quit
		win.OnClose(func(e *mygo.CloseEvent) {
			if !s.closeWindow() {
				e.PreventDefault()
			}
		})
		win.OnReadyToShow(func() {
			bounds := win.ContentBounds()
			if bounds.Width < minContentWidth || bounds.Height < minContentHeight {
				win.SetContentSize(max(bounds.Width, minContentWidth), max(bounds.Height, minContentHeight))
			}
			width, height := win.Size()
			win.SetMinimumSize(minContentWidth+width-bounds.Width, minContentHeight+height-bounds.Height)
		})
		mygo.App.OnSecondInstance(func([]string, string) { s.wake() })
		s.notify = func(body string) {
			if err := mygo.NewNotification(mygo.NotificationOptions{Title: view.Title(), Body: body}).Show(); err != nil {
				log.Printf("show the tray balloon: %v", err)
			}
		}
		tray, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    trayIcon,
			ToolTip: view.Title(),
			Menu:    trayMenu(s),
		})
		if err != nil {
			log.Printf("no tray icon: %v", err)
		} else {
			s.tray = true
			tray.OnClick(func() { s.toggle() })
			s.retray = func() {
				tray.SetToolTip(view.Title())
				tray.SetMenu(trayMenu(s))
			}
		}
		s.followSystem()
		s.startCompat()
		go s.watch(steam.PollInterval)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func trayMenu(s *session) *mygo.Menu {
	lang := s.resolvedLanguage()
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: view.Show(lang), Click: func(*mygo.MenuItem, *mygo.Window) { s.wake() }},
		{Label: view.RestartSteam(lang), Click: func(*mygo.MenuItem, *mygo.Window) { go s.restartSteam() }},
		{Label: view.OpenLuaFolder(lang), Disabled: !s.factsWithoutGate().SteamPathValid, Click: func(*mygo.MenuItem, *mygo.Window) { go s.openLuaFolder() }},
		{Label: view.Exit(lang), Click: func(*mygo.MenuItem, *mygo.Window) { s.requestExit() }},
	})
}
