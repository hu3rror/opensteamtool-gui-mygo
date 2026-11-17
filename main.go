package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
	"opensteamtool-gui-mygo/internal/view"
)

type window struct {
	facts app.Facts
}

func (w *window) view(c *ui.Context) {
	view.Main(c, app.Derive(w.facts))
}

func startupLang() app.Lang {
	langs := mygo.App.PreferredLanguages()
	if len(langs) == 0 {
		return app.English
	}
	return app.LangFor(langs[0])
}

func factsFor(lang app.Lang, steamPath string, host rt.Runtime) app.Facts {
	facts := app.Facts{Lang: lang, SteamPathValid: host.IsDir(steamPath)}
	if facts.SteamPathValid {
		facts.Deployment = app.DeploymentOf(app.OpenSteamToolLayout(), "", steamPath, host)
	}
	return facts
}

func main() {
	w := &window{facts: factsFor(startupLang(), "", rt.OS{})}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:     view.Title(w.facts.Lang),
			Width:     480,
			Height:    360,
			MinWidth:  320,
			MinHeight: 280,
			StateKey:  "main",
			Content:   ui.View(w.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
