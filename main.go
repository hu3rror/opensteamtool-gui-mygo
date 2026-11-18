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
	dir := toolDir()
	cfg, readable, err := config.Load(dir, rt.OS{})
	if err != nil {
		log.Printf("read config: %v", err)
	}
	s := newSession(rt.OS{}, time.Sleep, locale(), cfg, readable, dir)
	s.persist = func(cfg config.Config) error { return config.Save(dir, cfg) }

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     view.Title(s.resolvedLanguage()),
			Width:     480,
			Height:    360,
			MinWidth:  320,
			MinHeight: 280,
			StateKey:  "main",
			Content:   ui.View(s.view),
		})
		s.frame = win.Invalidate
		s.onUI = win.Update
		s.retitle = win.SetTitle
		lang := s.resolvedLanguage()
		if _, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    trayIcon,
			ToolTip: view.Title(lang),
			Menu: mygo.NewMenu([]*mygo.MenuItem{{
				Label: view.RestartSteam(lang),
				Click: func(*mygo.MenuItem, *mygo.Window) { go s.restartSteam() },
			}}),
		}); err != nil {
			log.Printf("no tray icon: %v", err)
		}
		go s.watch(steam.PollInterval)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
