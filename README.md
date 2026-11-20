<div align="center">
  <img src="resources/icon.png" width="120" alt="OpenSteamTool Manager icon">

  <h1>OpenSteamTool Manager</h1>

  <p><em>Native Windows app that applies the OpenSteamTool patch to Steam and keeps it in step with your Steam build</em></p>

  <p>
    <a href="https://github.com/hu3rror/opensteamtool-gui-mygo/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/hu3rror/opensteamtool-gui-mygo/ci.yml?style=flat-square&label=CI" alt="CI"></a>
    <img src="https://img.shields.io/badge/platform-Windows-0078d4?style=flat-square&logo=windows&logoColor=white" alt="Windows">
    <img src="https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.27">
    <a href="https://github.com/egoist/mygo"><img src="https://img.shields.io/badge/UI-MyGo-5b4ee9?style=flat-square" alt="MyGo"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT"></a>
  </p>

  <p><a href="README.md">English</a> · <a href="README_zh-CN.md">简体中文</a></p>
</div>

OpenSteamTool Manager is a desktop front end for the [OpenSteamTool](https://github.com/OpenSteam001/OpenSteamTool) patch. You pick a patch source; the app downloads that source's release, deploys the files into your Steam installation, and starts Steam again. It also removes the patch, tells you whether the patch fits the Steam build you have, and keeps the files you changed by hand.

The window is native — Go and [MyGo](https://github.com/egoist/mygo), no web view, no bundled runtime.

## Features

- **Apply in one click** — closes Steam when it is running, deploys the downloaded patch, then starts Steam. A second action removes the patch the same way.
- **Patch sources by repository** — three built-in sources plus any `owner/repo` on GitHub. Each source is locked to the one file layout it publishes, so switching sources never mixes files from two layouts.
- **Patch updates** — checks the latest release of the selected source, normalises versions (`v1.2.3` and `1.2.3` are the same), and keeps one cached copy per source.
- **Compatibility check** — reads the installed Steam build and reports whether the patch supports it, with an offline cache you can prewarm so a later check needs no network.
- **Injection entry** — chooses which proxy DLL loads the patch; the alternate entry of the OpenSteamTool layout is available for troubleshooting.
- **First-run wizard** — language, theme, Steam path and the initial patch download in four steps. It appears on its own when the configuration is missing or the saved path is invalid.
- **Steam path detection** — reads `SteamPath` from `HKCU\Software\Valve\Steam` and `InstallPath` from both `HKLM` Valve keys, accepts a candidate only when `steam.exe` is there, and falls back to the usual `Program Files` locations. Anything else, you pick by hand.
- **Chinese and English UI**, dark and light themes, both following the system by default.
- **Tray resident** — closing the window hides it; quitting from the tray asks first.

## Requirements

| | |
|---|---|
| System | Windows, x64 (the app builds for Windows only) |
| Steam | Installed and started at least once, so the client files exist |
| Network | Needed for the first patch download and for checks; a cached patch and a prewarmed compatibility cache both work offline |

## Install

No release is published yet, so build it yourself or take the artifact of a Release workflow run:

```bash
go tool mygo build
```

`build/windows-amd64/` then holds `opensteamtool-gui-mygo.exe` and the installer `opensteamtool-gui-mygo-setup-<version>-amd64.exe`. The folder is portable: the executable writes its caches next to itself, so an installed copy and a copied folder behave the same.

## First run

The wizard asks for the interface language, the theme, the Steam folder and the first patch download. It skips itself once a valid configuration exists, and **Settings → About → Run the setup wizard again** brings it back with the current values as the starting point.

The main window then shows one of two facts: **Applied** or **Not applied**, plus the Steam client state, the patch version and a Steam core compatibility line. When Steam is running, the primary action offers to close it before deploying.

## Patch sources

| Source | Repository | Layout | Note |
|---|---|---|---|
| Official | [`OpenSteam001/OpenSteamTool`](https://github.com/OpenSteam001/OpenSteamTool) | OpenSteamTool | Not updated for a long time |
| Community-maintained | [`huanyuejue/OpenSteamTool`](https://github.com/huanyuejue/OpenSteamTool) | OpenSteamTool | The default source |
| SteamDaddy | [`Contrary-v2/SteamDaddy-Dll`](https://github.com/Contrary-v2/SteamDaddy-Dll) | SteamDaddy | Different layout, different injection entry |
| Custom | any `owner/repo` on GitHub | OpenSteamTool | Must publish a `*-Release.zip` asset |

Hovering a source row shows its role, layout and note. The layout is not a setting: it follows from the source, because a patch release only makes sense in its own layout.

## Settings

| Tab | Holds |
|---|---|
| General | Interface language, theme |
| Patch source | Source choice, custom repository, injection entry, patch update check |
| Steam | Steam installation path, automatic detection, `steam.cfg` note |
| Lua configuration | Opens the Lua folder of the current layout |
| About | App version, app update check, project page, setup wizard |

## Files the app keeps

Everything stays in the folder that holds the executable, so removing that folder leaves nothing behind:

| Path | Content |
|---|---|
| `patches/<source>/` | The cached patch of that source, one version at a time |
| `cache/verified.toml` | Component fingerprints that passed verification |
| `userdata/<source>/` | Optional files you changed by hand, restored on the next deploy |

The compatibility cache lives in the Steam folder, under `opensteamtool/`, and is regenerated when missing.

## Development

Go 1.27 or newer, pinned by `go.mod` and `mise.toml`. The MyGo CLI is a tool of the module, so `go tool mygo` always runs the version `go.mod` pins.

```bash
mise install
go tool mygo dev                          # development app, restarts on change
go tool mygo vet .                        # go vet plus MyGo's UI checks
CGO_ENABLED=0 go test ./...               # the suite
go tool mygo build                        # build/<os>-<arch>/
```

`.github/workflows/ci.yml` runs vet and the tests on `windows-latest`. `.github/workflows/release.yml` builds the Windows artifacts on a `v*` tag and uploads them as a workflow artifact; it does not publish a release.

The code is split by concern: `internal/app` holds the state, `internal/view` the window, `internal/patch`, `internal/deploy`, `internal/source`, `internal/compat` and `internal/steam` the domain. `GLOSSARY.md` fixes the vocabulary, `docs/adr/` the decisions behind it.

## Notes

> [!WARNING]
> This is an unofficial tool. It modifies the Steam client and is not affiliated with Valve. Whether that is allowed is between you and Steam's Subscriber Agreement.

> [!NOTE]
> The app does not update itself yet. **About → Check for app updates** compares the running version with the latest GitHub release and offers the download page.

The patch binaries belong to their own projects and are licensed GPL-3.0; this app downloads them at run time and ships none of them. Its own source is MIT — see [`LICENSE`](LICENSE).
