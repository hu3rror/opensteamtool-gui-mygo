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

func text(l app.Lang, k key) string {
	if l == app.English {
		if s, ok := copyTable[k]; ok && s != "" {
			return s
		}
	}
	return string(k)
}

func Title(l app.Lang) string { return text(l, windowTitle) }
