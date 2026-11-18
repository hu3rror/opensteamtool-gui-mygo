package view

import (
	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/source"
)

type key string

const (
	applied                  key = "已应用"
	notApplied               key = "未应用"
	patchFailed              key = "补丁应用失败"
	applying                 key = "正在应用补丁…"
	removing                 key = "正在卸载补丁…"
	startingSteam            key = "正在启动 Steam…"
	stoppingSteam            key = "正在关闭 Steam…"
	invalidPath              key = "Steam 路径无效"
	invalidPathHint          key = "请在设置中指定有效的 Steam 安装路径"
	patchNotDownloaded       key = "补丁未下载"
	downloadHint             key = "请点击「检查补丁更新」下载新版本"
	windowTitle              key = "OpenSteamTool 一键管理工具"
	steamRunning             key = "Steam 正在运行"
	steamStopped             key = "Steam 未运行"
	restartSteam             key = "重启 Steam"
	steamStarted             key = "Steam 已启动"
	steamRestarted           key = "Steam 已重启"
	steamMissing             key = "未找到 steam.exe"
	closeSteamFailed         key = "关闭 Steam 失败"
	startSteamFailed         key = "启动 Steam 失败"
	settings                 key = "设置"
	tabGeneral               key = "通用"
	tabPatchSource           key = "补丁源"
	tabSteam                 key = "Steam 设置"
	tabLua                   key = "Lua 配置"
	tabAbout                 key = "关于"
	languageRow              key = "语言"
	languageAuto             key = "跟随系统"
	languageChinese          key = "简体中文"
	languageEnglish          key = "英语"
	steamPathRow             key = "Steam 安装路径"
	steamPathHint            key = "选择 steam.exe 所在的文件夹"
	steamPathError           key = "目录不存在或无法读取"
	browse                   key = "浏览…"
	closeButton              key = "关闭"
	steamCfgNote             key = "steam.cfg 已锁定 Steam 客户端自动更新"
	steamDialogTitle         key = "选择 Steam 目录"
	setupNotice              key = "尚未完成设置"
	setupHint                key = "请在设置中选择 Steam 安装目录"
	openSettings             key = "打开设置"
	officialSource           key = "官方源（长期未更新）"
	communitySource          key = "社区维护"
	steamDaddySource         key = "社区维护（SteamDaddy 布局）"
	customRepository         key = "自定义仓库"
	customSourceHint         key = "自定义源需 github.com 仓库地址（owner/repo）"
	fileLayout               key = "文件布局"
	openSteamToolLayoutLabel key = "OpenSteamTool 布局"
	steamDaddyLayoutLabel    key = "SteamDaddy 布局"
	entryRow                 key = "注入入口"
	entryDefault             key = "默认"
	entryEarly               key = "早期隔离"
	applyAndDownload         key = "应用并下载"
	addressError             key = "地址无效：需要 github.com 仓库地址"
)

var copyTable = map[key]string{
	applied:                  "Applied",
	notApplied:               "Not applied",
	patchFailed:              "Patch failed to apply",
	applying:                 "Applying the patch…",
	removing:                 "Removing the patch…",
	startingSteam:            "Starting Steam…",
	stoppingSteam:            "Closing Steam…",
	invalidPath:              "Steam path is invalid",
	invalidPathHint:          "Set a valid Steam installation path in settings",
	patchNotDownloaded:       "Patch not downloaded",
	downloadHint:             "Click Check for patch updates to download a new version",
	windowTitle:              "OpenSteamTool Manager",
	steamRunning:             "Steam is running",
	steamStopped:             "Steam is not running",
	restartSteam:             "Restart Steam",
	steamStarted:             "Steam started",
	steamRestarted:           "Steam restarted",
	steamMissing:             "steam.exe not found",
	closeSteamFailed:         "Failed to close Steam",
	startSteamFailed:         "Failed to start Steam",
	settings:                 "Settings",
	tabGeneral:               "General",
	tabPatchSource:           "Patch source",
	tabSteam:                 "Steam",
	tabLua:                   "Lua configuration",
	tabAbout:                 "About",
	languageRow:              "Language",
	languageAuto:             "Follow system",
	languageChinese:          "Chinese (simplified)",
	languageEnglish:          "English",
	steamPathRow:             "Steam installation path",
	steamPathHint:            "Choose the folder that holds steam.exe",
	steamPathError:           "The directory does not exist or cannot be read",
	browse:                   "Browse…",
	closeButton:              "Close",
	steamCfgNote:             "steam.cfg already locks the Steam client's automatic updates",
	steamDialogTitle:         "Choose the Steam directory",
	setupNotice:              "Setup is not complete",
	setupHint:                "Choose the Steam installation folder in the settings",
	openSettings:             "Open settings",
	officialSource:           "Official (not updated for a long time)",
	communitySource:          "Community-maintained",
	steamDaddySource:         "Community-maintained (SteamDaddy layout)",
	customRepository:         "Custom repository",
	customSourceHint:         "A custom source needs a github.com repository address (owner/repo)",
	fileLayout:               "File layout",
	openSteamToolLayoutLabel: "OpenSteamTool layout",
	steamDaddyLayoutLabel:    "SteamDaddy layout",
	entryRow:                 "Injection entry",
	entryDefault:             "Default",
	entryEarly:               "Early isolation",
	applyAndDownload:         "Apply and download",
	addressError:             "Invalid address: a github.com repository address is required",
}

var mainStateKeys = map[app.MainState]key{
	app.MainApplied:       applied,
	app.MainNotApplied:    notApplied,
	app.MainPatchFailed:   patchFailed,
	app.MainApplying:      applying,
	app.MainRemoving:      removing,
	app.MainStartingSteam: startingSteam,
	app.MainStoppingSteam: stoppingSteam,
}
var tabCopies = map[app.Tab]key{
	app.TabGeneral:     tabGeneral,
	app.TabPatchSource: tabPatchSource,
	app.TabSteam:       tabSteam,
	app.TabLua:         tabLua,
	app.TabAbout:       tabAbout,
}

func tabLabels(l app.Lang) []string {
	tabs := app.Tabs()
	labels := make([]string, len(tabs))
	for i, tab := range tabs {
		labels[i] = text(l, tabCopies[tab])
	}
	return labels
}

type explanation struct{ title, body key }

var explanationCopies = map[app.Explanation]explanation{
	app.InvalidSteamPath:   {invalidPath, invalidPathHint},
	app.PatchNotDownloaded: {patchNotDownloaded, downloadHint},
}

func steamWord(running bool) key {
	if running {
		return steamRunning
	}
	return steamStopped
}

var sourceLabels = map[string]key{
	source.KeyOfficial:   officialSource,
	source.KeyCommunity:  communitySource,
	source.KeySteamDaddy: steamDaddySource,
}

var layoutLabels = map[source.Layout]key{
	source.LayoutOpenSteamTool: openSteamToolLayoutLabel,
	source.LayoutSteamDaddy:    steamDaddyLayoutLabel,
}

var entryLabels = map[source.Entry]key{
	source.EntryDefault: entryDefault,
	source.EntryEarly:   entryEarly,
}

var resultCopies = map[app.Result]key{
	app.SteamStarted:        steamStarted,
	app.SteamRestarted:      steamRestarted,
	app.MissingSteamProgram: steamMissing,
	app.CloseSteamFailed:    closeSteamFailed,
	app.StartSteamFailed:    startSteamFailed,
}

func text(l app.Lang, k key) string {
	if l == app.English {
		if s, ok := copyTable[k]; ok && s != "" {
			return s
		}
	}
	return string(k)
}

func Title(l app.Lang) string { return text(l, windowTitle) }

func RestartSteam(l app.Lang) string   { return text(l, restartSteam) }
func SteamDirectory(l app.Lang) string { return text(l, steamDialogTitle) }
