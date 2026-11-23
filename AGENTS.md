# opensteamtool-gui-mygo

Project conventions for agents working in this repository.

## Toolchain and commands

The app is native UI: Go 1.27 from `mise.toml` (`go.mod` requires 1.27.4), no frontend, no Bun.

The MyGo CLI is a tool of `go.mod`, so `go tool mygo` runs the version the module pins and prints its own command list with `-h`.

- Check: `go tool mygo vet .` — Go vet plus MyGo's type-aware checks for UI build values kept in structs, package variables or goroutines. Then `CGO_ENABLED=0 go test ./...`.
- Run: `go tool mygo dev` — builds a development app and restarts it when a Go file, the configuration, the icon or a resource changes.
- Package: `go tool mygo build` — writes the apps to `build/<os>-<arch>`, the `out` of `mygo.json`.
- Refresh MyGo's own skills: `go tool mygo install-skills` — overwrites `mygo-maintenance` and `mygo-native-modules` in `.agents/skills` and keeps the others.

## Widgets and layout

Controls come from `breeze`, `github.com/yldm-tech/mygo-breeze-ui/ui/breeze`; MyGo's `github.com/egoist/mygo/ui` supplies the layout — `Row`, `Column`, `Box`, `Spacer`, `Scroll` and the style chains — plus the drawing atoms `breeze` also uses (`ui.Icon`, `ui.MustParseSVG`), the theme scope `ui.Themed` and the look-less `ui.RadioBase` that makes a selectable card. No screen draws itself with `ui`'s controls, and no control is drawn by hand. See `docs/adr/0007-widget-layer.md`.

## Agent skills

### Issue tracker

Issues live in this repo's GitHub Issues, via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical roles use the default label strings. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
