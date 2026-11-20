<div align="center">
  <img src="resources/icon.png" width="120" alt="OpenSteamTool Manager 图标">

  <h1>OpenSteamTool 一键管理工具</h1>

  <p><em>给 Steam 打上 OpenSteamTool 补丁、并让补丁跟着 Steam 版本走的原生 Windows 应用</em></p>

  <p>
    <a href="https://github.com/hu3rror/opensteamtool-gui-mygo/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/hu3rror/opensteamtool-gui-mygo/ci.yml?style=flat-square&label=CI" alt="CI"></a>
    <img src="https://img.shields.io/badge/platform-Windows-0078d4?style=flat-square&logo=windows&logoColor=white" alt="Windows">
    <img src="https://img.shields.io/badge/Go-1.27-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.27">
    <a href="https://github.com/egoist/mygo"><img src="https://img.shields.io/badge/UI-MyGo-1B7A5E?style=flat-square" alt="MyGo"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT"></a>
  </p>

  <p><a href="README.md">English</a> · <a href="README_zh-CN.md">简体中文</a></p>
</div>

这是 [OpenSteamTool](https://github.com/OpenSteam001/OpenSteamTool) 补丁的桌面外壳。选一个补丁源，工具会下载该源的最新发布、把文件部署进你的 Steam 目录、再把 Steam 启动起来；也可以反过来把补丁移除。除此之外，它告诉你补丁和你当前的 Steam 版本合不合得来，并替你保留手工改过的文件。

窗口是原生的：Go 加 [MyGo](https://github.com/egoist/mygo)，没有 WebView，也不打包运行时。

## 功能

- **一键应用** — Steam 在跑就先关掉，部署下载好的补丁，再启动 Steam；卸载动作则会先问一句要不要关掉 Steam。
- **以仓库地址识别补丁源** — 三个内置源，外加任意 GitHub `owner/repo`。每个源锁定它自己发布的文件布局，切换源不会把两种布局的文件混在一起。
- **补丁更新** — 检查所选源的最新发布，版本号归一化处理（`v1.2.3` 与 `1.2.3` 视为同一版），每个源只留一份缓存。
- **兼容体检** — 读出已安装的 Steam 构建号，判断补丁是否覆盖它；结果可缓存，缓存预热之后不联网也能出结论。
- **注入入口** — 决定由哪个代理 DLL 加载补丁；OpenSteamTool 布局的早期隔离入口留给排障，官方源下不可选。
- **首次运行向导** — 界面语言、主题、Steam 路径、首次补丁下载，四步走完。配置缺失或保存的路径失效时，它会自己出现。
- **Steam 路径自动检测** — 从 `HKCU\Software\Valve\Steam` 的 `SteamPath`、以及两个 `HKLM` Valve 键的 `InstallPath` 取候选，只有目录里真有 `steam.exe` 才采纳，最后回退到常见的 `Program Files` 位置。都没有，就让你手动选。
- **中文与英文界面**，深色与浅色主题，默认都跟随系统。
- **常驻托盘** — 关闭窗口只是隐藏；退出托盘前会再确认一次。

## 运行环境

| | |
|---|---|
| 系统 | Windows，x64（本项目只构建 Windows 版） |
| Steam | 至少安装并启动过一次，客户端文件存在即可 |
| 网络 | 首次下载补丁与各类检查需要；补丁已缓存、体检缓存已预热时，可以离线使用 |

## 安装

目前还没有发布任何 release，因此二选一：自己构建，或取 Release 工作流的产物。

```bash
go tool mygo build
```

`build/windows-amd64/` 里会得到 `opensteamtool-gui-mygo.exe` 与安装包 `opensteamtool-gui-mygo-setup-<版本>-amd64.exe`。这个目录是绿色的：可执行文件把缓存写在自己旁边，安装版和直接拷贝的目录行为一致。

## 首次使用

向导依次问界面语言、主题、Steam 目录、首次补丁下载。配置有效后它不再出现；**设置 → 关于 → 重新运行向导** 可以随时叫回，并把当前值当作起点。

主窗口只有两种状态要看清：**已应用** 或 **未应用**。下方一行给出 Steam 客户端状态、上一次操作的结果，以及补丁更新检查。Steam 核心兼容性整段在 **设置 → Steam 设置** 里；补丁和你的 Steam 版本合不来时，主窗口会多一行警告，点它就到那里。Steam 正在运行时，应用动作直接关掉 Steam，只有卸载会先问。

## 补丁源

| 源 | 仓库 | 布局 | 备注 |
|---|---|---|---|
| 官方源 | [`OpenSteam001/OpenSteamTool`](https://github.com/OpenSteam001/OpenSteamTool) | OpenSteamTool | 长期未更新 |
| 社区维护源 | [`huanyuejue/OpenSteamTool`](https://github.com/huanyuejue/OpenSteamTool) | OpenSteamTool | 新装默认 |
| SteamDaddy 源 | [`Contrary-v2/SteamDaddy-Dll`](https://github.com/Contrary-v2/SteamDaddy-Dll) | SteamDaddy | 布局不同，注入入口也不同 |
| 自定义源 | 任意 GitHub `owner/repo` | OpenSteamTool | 发布里最好带 `*-Release.zip`，其他 `.zip` 也会被采用 |

鼠标停在某个源上，会显示它的角色和布局，官方源还带备注。布局不是设置项：它跟着源走，因为一份补丁发布只在自己的布局里成立。

## 设置

| 页 | 内容 |
|---|---|
| 通用 | 界面语言、主题 |
| 补丁源 | 补丁源选择、自定义仓库，以及草稿与当前源不同时的 **应用并下载** |
| Steam 设置 | Steam 安装路径与自动检测、注入入口、`steam.cfg` 说明、Steam 核心兼容性 |
| Lua 配置 | 目前为空页；Lua 目录从托盘菜单打开 |
| 关于 | 软件版本、应用更新检查、项目主页、重新运行向导 |

## 工具留下的文件

配置与缓存都在可执行文件所在目录，删掉这个目录不会留下它自己的东西：

| 路径 | 内容 |
|---|---|
| `patches/<源>/` | 该源的补丁缓存，同一时间一份版本 |
| `cache/verified.toml` | 通过校验的组件指纹 |
| `userdata/<源>/` | 你手工改过的可选文件，下次部署时还原 |

体检缓存放在 Steam 目录下的 `opensteamtool/`，缺失时重新生成。

## 开发

需要 Go 1.27 或更高，版本由 `go.mod` 与 `mise.toml` 锁定。MyGo 命令行是该模块的工具，`go tool mygo` 永远跑 `go.mod` 里钉住的那个版本。

```bash
mise install
go tool mygo dev                          # 开发版应用，改文件即重启
go tool mygo vet .                        # go vet 加 MyGo 的 UI 检查
CGO_ENABLED=0 go test ./...               # 测试套件
go tool mygo build                        # 输出到 build/<os>-<arch>/
```

`.github/workflows/ci.yml` 在 `windows-latest` 上跑 vet 与测试。`.github/workflows/release.yml` 在 `v*` 标签上构建 Windows 产物并作为 workflow artifact 上传，不发布 release。

代码按职责拆分：`internal/app` 管状态，`internal/view` 管窗口，`internal/patch`、`internal/deploy`、`internal/source`、`internal/compat`、`internal/steam` 管领域。术语表在 `GLOSSARY.md`，决策记录在 `docs/adr/`。

## 说明

> [!WARNING]
> 这是非官方工具，会修改 Steam 客户端，与 Valve 无关。是否允许这么做，由你和 Steam 订户协议决定。

> [!NOTE]
> 应用还不会自动更新自己。**关于 → 检查应用更新** 把当前版本与 GitHub 最新 release 比对，然后给出下载页。

补丁的二进制属于各自的项目，采用 GPL-3.0；本工具在运行时下载它们，自身不携带。本工具的源码是 MIT，见 [`LICENSE`](LICENSE)。
