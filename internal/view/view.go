package view

import (
	"github.com/egoist/mygo/ui"

	"opensteamtool-gui-mygo/internal/app"
)

func Main(c *ui.Context, v app.View) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Gap(t.Space(2)).Children(func() {
		ui.Column(c).Fill().Gap(t.Space(2)).Children(func() {
			ui.Text(c, text(v.Lang, mainStateKeys[v.MainState])).Role(ui.RoleStatus).FontSize(t.FontSize * 1.5)
			if copy, ok := explanationCopies[v.Explanation]; ok {
				ui.Text(c, text(v.Lang, copy.title))
				ui.Text(c, text(v.Lang, copy.body)).TextColor(t.TextMuted)
			}
		})
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			ui.Text(c, text(v.Lang, steamWord(v.Status.SteamRunning))).Role(ui.RoleStatus)
			if stage, ok := app.Stage(v.Status.Busy); ok {
				ui.Text(c, text(v.Lang, mainStateKeys[stage])).Role(ui.RoleStatus)
			}
			if copy, ok := resultCopies[v.Status.Result]; ok {
				ui.Text(c, text(v.Lang, copy)).Role(ui.RoleStatus)
			}
		})
	})
}
