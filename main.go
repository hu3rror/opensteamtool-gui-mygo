package main

import (
	_ "embed"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/steam"
	"opensteamtool-gui-mygo/internal/view"
)

//go:embed resources/icon.png
var trayIcon []byte

func startupLang() app.Lang {
	langs := mygo.App.PreferredLanguages()
	if len(langs) == 0 {
		return app.English
	}
	return app.LangFor(langs[0])
}

func factsFor(lang app.Lang, steamPath string, host rt.Runtime) app.Facts {
	facts := app.Facts{Lang: lang, SteamPathValid: host.IsDir(steamPath)}
	if running, err := steam.Running(host, steamPath); err == nil {
		facts.SteamRunning = running
	}
	if facts.SteamPathValid {
		facts.Deployment = app.DeploymentOf(app.OpenSteamToolLayout(), "", steamPath, host)
	}
	return facts
}

func main() {
	s := newSession(startupLang(), "", rt.OS{}, time.Sleep)
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     view.Title(s.snapshot().Lang),
			Width:     480,
			Height:    360,
			MinWidth:  320,
			MinHeight: 280,
			StateKey:  "main",
			Content:   ui.View(s.view),
		})
		s.frame = win.Invalidate
		facts := s.snapshot()
		if _, err := mygo.NewTray(mygo.TrayOptions{
			Icon:    trayIcon,
			ToolTip: view.Title(facts.Lang),
			Menu: mygo.NewMenu([]*mygo.MenuItem{{
				Label: view.RestartSteam(facts.Lang),
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
