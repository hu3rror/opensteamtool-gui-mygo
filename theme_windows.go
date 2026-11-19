//go:build windows

package main

import "golang.org/x/sys/windows/registry"

// systemAppearance reads the desktop's light or dark preference. The second
// value is false where Windows does not answer, which the theme reads as dark.
func systemAppearance() (bool, bool) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false, false
	}
	defer func() { _ = key.Close() }()
	light, _, err := key.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false, false
	}
	return light == 0, true
}
