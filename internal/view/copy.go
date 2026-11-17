package view

import "opensteamtool-gui-mygo/internal/app"

type key string

const (
	applied            key = "已应用"
	notApplied         key = "未应用"
	patchFailed        key = "补丁应用失败"
	applying           key = "正在应用补丁…"
	removing           key = "正在卸载补丁…"
	startingSteam      key = "正在启动 Steam…"
	stoppingSteam      key = "正在关闭 Steam…"
	invalidPath        key = "Steam 路径无效"
	invalidPathHint    key = "请在设置中指定有效的 Steam 安装路径"
	patchNotDownloaded key = "补丁未下载"
	downloadHint       key = "请点击「检查补丁更新」下载新版本"
	windowTitle        key = "OpenSteamTool 一键管理工具"
	steamRunning       key = "Steam 正在运行"
	steamStopped       key = "Steam 未运行"
	restartSteam       key = "重启 Steam"
	steamStarted       key = "Steam 已启动"
	steamRestarted     key = "Steam 已重启"
	steamMissing       key = "未找到 steam.exe"
	closeSteamFailed   key = "关闭 Steam 失败"
	startSteamFailed   key = "启动 Steam 失败"
)

var copyTable = map[key]string{
	applied:            "Applied",
	notApplied:         "Not applied",
	patchFailed:        "Patch failed to apply",
	applying:           "Applying the patch…",
	removing:           "Removing the patch…",
	startingSteam:      "Starting Steam…",
	stoppingSteam:      "Closing Steam…",
	invalidPath:        "Steam path is invalid",
	invalidPathHint:    "Set a valid Steam installation path in settings",
	patchNotDownloaded: "Patch not downloaded",
	downloadHint:       "Click Check for patch updates to download a new version",
	windowTitle:        "OpenSteamTool Manager",
	steamRunning:       "Steam is running",
	steamStopped:       "Steam is not running",
	restartSteam:       "Restart Steam",
	steamStarted:       "Steam started",
	steamRestarted:     "Steam restarted",
	steamMissing:       "steam.exe not found",
	closeSteamFailed:   "Failed to close Steam",
	startSteamFailed:   "Failed to start Steam",
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

func RestartSteam(l app.Lang) string { return text(l, restartSteam) }
