package app

import "strings"

type ToolState int

const (
	ToolUnchecked ToolState = iota
	ToolChecking
	ToolUpToDate
	ToolAvailable
	ToolCheckFailed
)

type ToolCheck struct {
	Version   string
	State     ToolState
	Available string
}

func toolCheckOf(f Facts) ToolCheck {
	check := ToolCheck{Version: f.ToolVersion}
	switch {
	case f.Busy == BusyCheckTool:
		check.State = ToolChecking
	case f.ToolFailed:
		check.State = ToolCheckFailed
	case !f.ToolChecked:
		check.State = ToolUnchecked
	case ToolNewer(f.ToolOnline, f.ToolVersion):
		check.State = ToolAvailable
		check.Available = f.ToolOnline
	default:
		check.State = ToolUpToDate
	}
	return check
}

// ToolNewer reports whether the released tool version is newer than the local
// one, comparing dot segments as numbers: a missing segment and a segment
// without digits count as zero.
func ToolNewer(online string, local string) bool {
	released := strings.Split(online, ".")
	running := strings.Split(local, ".")
	for i := 0; i < len(released) || i < len(running); i++ {
		switch a, b := segmentNumber(released, i), segmentNumber(running, i); {
		case a > b:
			return true
		case a < b:
			return false
		}
	}
	return false
}

func segmentNumber(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	text := strings.TrimPrefix(strings.TrimPrefix(parts[i], "v"), "V")
	n := 0
	for _, r := range text {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
