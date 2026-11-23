<div align="center">
  <img src="resources/icon.png" width="120" alt="OpenSteamTool Manager 图标">

  <h1>OpenSteamTool Manager MyGo</h1>

  <p><em>给 Steam 打上 OpenSteamTool 补丁，并让补丁跟上 Steam 版本的原生 Windows 应用</em></p>

  <p>
    <a href="https://github.com/hu3rror/opensteamtool-gui-mygo/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/hu3rror/opensteamtool-gui-mygo/ci.yml?style=flat-square&label=CI" alt="CI"></a>
    <img src="https://img.shields.io/badge/platform-Windows-0078d4?style=flat-square&logo=windows&logoColor=white" alt="Windows">
    <a href="https://github.com/yldm-tech/mygo-breeze-ui"><img src="https://img.shields.io/badge/UI-Breeze-0067C0?style=flat-square" alt="Breeze UI"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT"></a>
  </p>

  <p><a href="README.md">English</a> · <a href="README_zh-CN.md">简体中文</a></p>
</div>

这是 [OpenSteamTool](https://github.com/OpenSteam001/OpenSteamTool) 补丁的桌面外壳。选一个补丁源，它下载这个源的最新发布，把文件装进你的 Steam 目录，再启动 Steam；反过来也能卸载。它还告诉你补丁跟当前的 Steam 版本合不合得来，并保留你手工改过的文件。

界面完全原生：Go 加 [MyGo](https://github.com/egoist/mygo)，控件用 [Breeze UI](https://github.com/yldm-tech/mygo-breeze-ui) 按 Windows 11 的画法绘制，没有 WebView，也不打包运行时。桌面的明暗、强调色、对比度和文字大小，控件都跟着走。

## 功能

- **一键应用** — Steam 开着就先关掉，部署下载好的补丁，再把 Steam 启动起来。卸载前会先问一句要不要关 Steam。
- **补丁源按仓库区分** — 内置几个源，也能填任意 GitHub `owner/repo`。每个源只对应它自己发布的文件布局，换源不会把两种布局的文件混在一起。
- **补丁更新** — 查所选源的最新发布，比较时 `v1.2.3` 和 `1.2.3` 算同一版；每个源只留一份缓存。
- **兼容体检** — 读出已安装的 Steam 构建号，判断补丁认不认它。结论会缓存，预热过之后断网也能给结果。
- **注入入口** — 决定哪个代理 DLL 加载补丁。OpenSteamTool 布局的早期隔离入口用来排障，官方源下选不到。
- **首次运行向导** — 依次问界面语言、主题、Steam 路径和首次补丁下载。配置缺失、或存下的路径失效时，它自己出现。
- **Steam 路径自动检测** — 先看 `HKCU\Software\Valve\Steam` 的 `SteamPath` 和两个 `HKLM` Valve 键的 `InstallPath`，目录里真有 `steam.exe` 才认；再找不到，就试常见的 `Program Files` 位置，最后仍没有就让你自己选。
- **中英双语界面**，深浅两套主题，默认跟随系统。
- **常驻托盘** — 关窗口只是隐藏；从托盘退出前还会问一次。

## 运行环境

| | |
|---|---|
| 系统 | Windows，x64（本项目只构建 Windows 版） |
| Steam | 至少安装并启动过一次，客户端文件存在即可 |
| 网络 | 首次下载补丁与各类检查需要；补丁已缓存、体检缓存已预热时，可以离线使用 |

## 安装

从 [最新 release](https://github.com/hu3rror/opensteamtool-gui-mygo/releases/latest) 下载：运行 `opensteamtool-gui-mygo-setup-<版本>-amd64.exe` 安装，或取 `opensteamtool-gui-mygo.exe` 放到任意目录直接运行。缓存写在可执行文件旁边，安装版与直接拷贝的目录行为一致。

自己构建：

```bash
go tool mygo build
```

`build/windows-amd64/` 里会得到 `opensteamtool-gui-mygo.exe`；NSIS 可用时还会得到安装包 `opensteamtool-gui-mygo-setup-<版本>-amd64.exe`（在 Windows 上 `mygo build` 第一次需要时会自动下载 NSIS）。

## 首次使用

向导依次问界面语言、主题、Steam 目录和首次补丁下载。配置有效后它不再出现；想再看，走 **设置 → 关于 → 重新运行向导**，当前的值会当作起点。

主窗口只会亮一个状态：**已应用**、**未应用**，或应用失败后的 **补丁应用失败**。下面一行写 Steam 客户端状态、上一次操作的结果和补丁更新检查。Steam 核心兼容性整段在 **设置 → Steam 设置** 里；补丁和你的 Steam 版本合不来时，主窗口会多一行警告，点它直接过去。Steam 开着时，应用会直接把它关掉，只有卸载会先问。

## 补丁源

| 源 | 仓库 | 布局 | 备注 |
|---|---|---|---|
| 官方源 | [`OpenSteam001/OpenSteamTool`](https://github.com/OpenSteam001/OpenSteamTool) | OpenSteamTool | 上游长期未更新；选中时应用会给出警告 |
| 社区维护源 | [`huanyuejue/OpenSteamTool`](https://github.com/huanyuejue/OpenSteamTool) | OpenSteamTool | 默认源 |
| SteamDaddy 源 | [`Contrary-v2/SteamDaddy-Dll`](https://github.com/Contrary-v2/SteamDaddy-Dll) | SteamDaddy | 布局不同，注入入口也不同 |
| 自定义源 | 任意 GitHub `owner/repo` | OpenSteamTool | 发布里最好带 `*-Release.zip`，其他 `.zip` 也会被采用 |

鼠标停在某个源上会显示它的角色和布局，官方源还多一条备注；选中官方源时，列表下面另有一行警告。布局不是设置项，它跟着源走——一份补丁发布只在自己的布局里成立。

## 设置

| 页 | 内容 |
|---|---|
| 通用 | 界面语言、主题 |
| 补丁源 | 选源、填自定义仓库；草稿和当前源不一样时出现 **应用并下载** |
| Steam 设置 | Steam 安装路径与自动检测、注入入口、`steam.cfg` 说明、Steam 核心兼容性 |
| Lua 目录 | 当前布局加载 `.lua` 脚本的目录，显示完整路径、`manifest.lua` 在不在，还有一个打开按钮；托盘菜单里也能打开 |
| 关于 | 软件版本、应用更新检查、项目主页、重新运行向导 |

## 工具留下的文件

配置和缓存都放在可执行文件旁边。删掉这个目录，它什么都不留：

| 路径 | 内容 |
|---|---|
| `config.json` | 你的设置：Steam 路径、界面语言、主题、补丁源、注入入口、上次部署 |
| `patches/<源>/` | 该源的补丁缓存，同一时间一份版本 |
| `cache/verified.toml` | 通过校验的组件指纹 |
| `userdata/<源>/` | 你手工改过的可选文件，下次部署时还原 |

体检缓存放在 Steam 目录下的 `opensteamtool/`，没了就重新生成。

## 开发

需要 Go 1.27 或更高，版本由 `go.mod` 和 `mise.toml` 锁住。MyGo 命令行就是这个模块的工具，`go tool mygo` 跑的是 `go.mod` 钉住的那个版本。

```bash
mise install
go tool mygo dev                          # 开发版应用，改文件即重启
go tool mygo vet .                        # go vet 加 MyGo 的 UI 检查
CGO_ENABLED=0 go test ./...               # 测试套件
go tool mygo build                        # 输出到 build/<os>-<arch>/
```

`.github/workflows/ci.yml` 在 `windows-latest` 上跑 vet 和测试。`.github/workflows/release.yml` 在 `v*` 标签上构建 Windows 产物，直接发布成 GitHub Release；标签要打在改过 `mygo.json` 版本号的那个提交上，安装包名里的版本号才对得上。

代码按职责拆分：`internal/app` 管状态，`internal/view` 管窗口，`internal/patch`、`internal/deploy`、`internal/source`、`internal/compat`、`internal/steam` 管领域。术语表在 `GLOSSARY.md`，决策记录在 `docs/adr/`。

## 说明

> [!WARNING]
> 这是非官方工具，会修改 Steam 客户端，与 Valve 无关。是否允许这么做，由你和 Steam 订户协议决定。

> [!IMPORTANT]
> 本工具只做补丁管理：不带 Lua 脚本，不带补丁文件，也不带任何破解或绕过手段，更不会替你去搜。它部署的补丁来自你选的源；Lua 脚本和其他扩展自己准备，用不用、合不合规，由你决定。

> [!NOTE]
> 应用不会自动更新自己。**关于 → 检查应用更新** 把当前版本与 GitHub 最新 release 比对，然后给出下载页。

补丁的二进制属于各自项目，按 GPL-3.0 授权；本工具只在运行时下载，自己不携带。工具本身的源码是 MIT，见 [`LICENSE`](LICENSE)。
