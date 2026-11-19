package view

import (
	"strings"

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
	showWindow               key = "显示"
	exitApp                  key = "退出"
	openLuaFolder            key = "打开 Lua 目录"
	trayRunning              key = "仍在托盘运行，点击图标可显示窗口"
	quitQuestion             key = "确定退出？进行中的操作会中止。"
	applyAndStart            key = "应用补丁并启动 Steam"
	removeAndDeploy          key = "移除并部署"
	uninstall                key = "卸载补丁"
	uninstallAndRestart      key = "卸载补丁并重启 Steam"
	exitAndUninstall         key = "退出 Steam 并卸载补丁"
	confirmTitle             key = "确认"
	runningQuestion          key = "Steam 正在运行。是否自动关闭 Steam 并继续？"
	yes                      key = "是"
	no                       key = "否"
	conflictFound            key = "检测到 %s，需先移除"
	steamStarted             key = "Steam 已启动"
	steamRestarted           key = "Steam 已重启"
	patchApplied             key = "补丁已应用"
	patchRemoved             key = "已卸载补丁"
	removalFailed            key = "卸载失败"
	steamMissing             key = "未找到 steam.exe"
	pathUnset                key = "请先指定有效的 Steam 安装路径"
	closeSteamFailed         key = "关闭 Steam 失败"
	startSteamFailed         key = "启动 Steam 失败"
	deploymentFailed         key = "部署失败"
	settings                 key = "设置"
	tabGeneral               key = "通用"
	tabPatchSource           key = "补丁源"
	tabSteam                 key = "Steam 设置"
	tabLua                   key = "Lua 配置"
	tabAbout                 key = "关于"
	languageRow              key = "语言"
	followSystem             key = "跟随系统"
	languageChinese          key = "简体中文"
	languageEnglish          key = "英语"
	themeRow                 key = "主题"
	themeDark                key = "深色"
	themeLight               key = "浅色"
	appVersionLabel          key = "软件版本"
	appUpdateRow             key = "应用更新"
	projectPage              key = "项目主页"
	checkToolUpdate          key = "检查应用更新"
	toolChecking             key = "正在检查应用更新..."
	toolUpToDate             key = "已是最新版本"
	toolAvailable            key = "发现新版本 %s"
	openDownloadPage         key = "打开下载页"
	runWizard                key = "重新运行向导"
	runWizardHint            key = "以当前语言、主题与 Steam 路径为初值重新运行设置向导。"
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
	checkUpdate              key = "检查补丁更新"
	downloadNewVersion       key = "下载并解压新版本"
	checking                 key = "检查中…"
	downloadingPatch         key = "正在下载补丁…"
	upToDate                 key = "补丁已是最新"
	newPatch                 key = "发现新补丁"
	checkFailed              key = "检查失败"
	clientUpdated            key = "Steam 客户端已更新，补丁可能不适配"
	downloaded               key = "新版本下载并解压完成"
	networkFailed            key = "网络请求失败"
	noAsset                  key = "发布包中没有符合 *-Release.zip 的资产"
	archivesOnly             key = "发布包只有非 zip 归档（.7z/.rar 等）"
	missingMember            key = "发布包缺少该源布局要求的文件"
	parseFailed              key = "解析线上版本失败"
	localIOFailed            key = "本地文件操作失败"
	writeFailed              key = "写入本地文件失败"
	compatSection            key = "Steam 核心兼容性"
	compatChecking           key = "正在检查兼容性..."
	compatCached             key = "完美兼容 (已缓存)"
	compatAdapted            key = "上游已适配 (未缓存)"
	compatOffline            key = "离线可用 (使用本地缓存)"
	compatNotAdapted         key = "上游尚未适配此版本"
	compatNoCore             key = "未找到核心 DLL"
	compatDown               key = "网络不可用"
	prewarmOffline           key = "预热离线缓存"
	compatDetails            key = "详细信息"
	cacheSignatures          key = "一键缓存签名"
	cachingSignatures        key = "正在缓存..."
	cacheReady               key = "缓存已就绪"
	cacheFailed              key = "缓存预热失败：%s"
	compatTipUpdated         key = "Steam 版本已更新，上游尚未发布适配签名，请等待更新。"
	compatTipCached          key = "核心特征码与 IPC 规约已全部就绪并离线缓存。"
	compatTipAdapted         key = "上游已适配，可一键预热离线缓存。"
	compatTipMissing         key = "未找到核心 DLL（steamclient64.dll / steamui.dll）。"
	compatTipOffline         key = "网络不可用，体检结果未知；已缓存项仍可离线使用。"
	riskNotAdapted           key = "当前 Steam 尚未适配"
	riskMissingCore          key = "未找到 Steam 核心文件"
	riskSettings             key = "设置 →"
	componentRow             key = "%s（%s）"
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
	showWindow:               "Show",
	exitApp:                  "Exit",
	openLuaFolder:            "Open Lua folder",
	trayRunning:              "Still running in the tray. Click the icon to show the window.",
	quitQuestion:             "Quit? Running operations will be aborted.",
	steamStarted:             "Steam started",
	steamRestarted:           "Steam restarted",
	patchApplied:             "Patch applied",
	patchRemoved:             "Patch removed",
	removalFailed:            "Removal failed",
	pathUnset:                "Set a valid Steam installation path first",
	deploymentFailed:         "Deployment failed",
	applyAndStart:            "Apply patch and start Steam",
	removeAndDeploy:          "Remove it and deploy",
	uninstall:                "Remove the patch",
	uninstallAndRestart:      "Remove the patch and restart Steam",
	exitAndUninstall:         "Exit Steam and remove the patch",
	confirmTitle:             "Confirm",
	runningQuestion:          "Steam is running. Close Steam and continue?",
	yes:                      "Yes",
	no:                       "No",
	conflictFound:            "Found %s, which must be removed first",
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
	followSystem:             "Follow system",
	languageChinese:          "Chinese (simplified)",
	languageEnglish:          "English",
	themeRow:                 "Theme",
	themeDark:                "Dark",
	themeLight:               "Light",
	appVersionLabel:          "App version",
	appUpdateRow:             "App update",
	projectPage:              "Project page",
	checkToolUpdate:          "Check for app updates",
	toolChecking:             "Checking for app updates...",
	toolUpToDate:             "Up to date",
	toolAvailable:            "Version %s is available",
	openDownloadPage:         "Open the download page",
	runWizard:                "Run the setup wizard again",
	runWizardHint:            "Run the setup wizard again, starting from the current language, theme and Steam path",
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
	checkUpdate:              "Check for patch updates",
	downloadNewVersion:       "Download and extract the new version",
	checking:                 "Checking…",
	downloadingPatch:         "Downloading the patch…",
	upToDate:                 "Patch is up to date",
	newPatch:                 "New patch found",
	checkFailed:              "Check failed",
	clientUpdated:            "Steam updated; the patch may not fit yet",
	downloaded:               "New version downloaded and extracted",
	networkFailed:            "Network request failed",
	noAsset:                  "The release has no *-Release.zip asset",
	archivesOnly:             "The release only contains non-zip archives (.7z, .rar)",
	missingMember:            "The release is missing files this layout requires",
	parseFailed:              "Failed to parse the online version",
	localIOFailed:            "Local file operation failed",
	writeFailed:              "Failed to write local files",
	compatSection:            "Steam core compatibility",
	compatChecking:           "Checking compatibility...",
	compatCached:             "Fully compatible (cached)",
	compatAdapted:            "Upstream has adapted (not cached)",
	compatOffline:            "Offline ready (using the local cache)",
	compatNotAdapted:         "Upstream has not adapted this build",
	compatNoCore:             "Core DLL not found",
	compatDown:               "Network unavailable",
	prewarmOffline:           "Cache for offline use",
	compatDetails:            "Details",
	cacheSignatures:          "Cache signatures",
	cachingSignatures:        "Caching...",
	cacheReady:               "Cache is ready",
	cacheFailed:              "Caching failed: %s",
	compatTipUpdated:         "Steam updated and upstream has not published an adaptation yet. Please wait.",
	compatTipCached:          "The core signature and IPC contract are ready and cached offline.",
	compatTipAdapted:         "Upstream has adapted. You can cache it for offline use in one click.",
	compatTipMissing:         "Core DLLs not found (steamclient64.dll / steamui.dll).",
	compatTipOffline:         "The network is unavailable, so the result is unknown. Cached items still work offline.",
	riskNotAdapted:           "The current Steam build is not supported yet",
	riskMissingCore:          "Steam core files not found",
	riskSettings:             "Settings →",
	componentRow:             "%s (%s)",
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
	app.PatchApplied:        patchApplied,
	app.PatchRemoved:        patchRemoved,
	app.RemovalFailed:       removalFailed,
	app.PathUnset:           pathUnset,
	app.DeploymentFailed:    deploymentFailed,
	app.MissingSteamProgram: steamMissing,
	app.CloseSteamFailed:    closeSteamFailed,
	app.StartSteamFailed:    startSteamFailed,
	app.Downloaded:          downloaded,
	app.NetworkFailed:       networkFailed,
	app.NoAsset:             noAsset,
	app.ArchivesOnly:        archivesOnly,
	app.MissingMember:       missingMember,
	app.ParseFailed:         parseFailed,
	app.LocalIOFailed:       localIOFailed,
	app.WriteFailed:         writeFailed,
}

var conclusionCopies = map[app.Conclusion]key{
	app.UpToDate:      upToDate,
	app.NewPatch:      newPatch,
	app.CheckFailed:   checkFailed,
	app.ClientUpdated: clientUpdated,
}

func text(l app.Lang, k key) string {
	if l == app.English {
		if s, ok := copyTable[k]; ok && s != "" {
			return s
		}
	}
	return string(k)
}

func conflictText(l app.Lang, name string) string {
	return strings.Replace(text(l, conflictFound), "%s", name, 1)
}

func toolAvailableText(l app.Lang, version string) string {
	return strings.Replace(text(l, toolAvailable), "%s", version, 1)
}

func Title(l app.Lang) string { return text(l, windowTitle) }

func RestartSteam(l app.Lang) string   { return text(l, restartSteam) }
func SteamDirectory(l app.Lang) string { return text(l, steamDialogTitle) }

func Show(l app.Lang) string          { return text(l, showWindow) }
func Exit(l app.Lang) string          { return text(l, exitApp) }
func OpenLuaFolder(l app.Lang) string { return text(l, openLuaFolder) }
func TrayRunning(l app.Lang) string   { return text(l, trayRunning) }

var healthCopies = map[app.Health]key{
	app.HealthChecking:           compatChecking,
	app.HealthCompatibleCached:   compatCached,
	app.HealthAdaptedNotCached:   compatAdapted,
	app.HealthOfflineReady:       compatOffline,
	app.HealthNotAdapted:         compatNotAdapted,
	app.HealthMissing:            compatNoCore,
	app.HealthNetworkUnavailable: compatDown,
}

var healthTips = map[app.Health]key{
	app.HealthCompatibleCached:   compatTipCached,
	app.HealthAdaptedNotCached:   compatTipAdapted,
	app.HealthNotAdapted:         compatTipUpdated,
	app.HealthMissing:            compatTipMissing,
	app.HealthNetworkUnavailable: compatTipOffline,
}

var riskCopies = map[app.Health]key{
	app.HealthNotAdapted: riskNotAdapted,
	app.HealthMissing:    riskMissingCore,
}

func HealthLabel(l app.Lang, health app.Health) string { return text(l, healthCopies[health]) }

func HealthTip(l app.Lang, health app.Health) (string, bool) {
	tip, ok := healthTips[health]
	if !ok {
		return "", false
	}
	return text(l, tip), true
}

func RiskLabel(l app.Lang, health app.Health) (string, bool) {
	risk, ok := riskCopies[health]
	if !ok {
		return "", false
	}
	return text(l, risk), true
}

func SettingsAffordance(l app.Lang) string { return text(l, riskSettings) }

func CacheFailure(l app.Lang, result app.Result) string {
	return strings.Replace(text(l, cacheFailed), "%s", text(l, resultCopies[result]), 1)
}

func ComponentLabel(l app.Lang, component app.Component) string {
	row := strings.Replace(text(l, componentRow), "%s", component.DLL, 1)
	return strings.Replace(row, "%s", component.Kind, 1)
}
