//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func registryString(keyPath, value string) string {
	root, name := registry.CURRENT_USER, keyPath
	if rest, ok := strings.CutPrefix(keyPath, `HKLM\`); ok {
		root, name = registry.LOCAL_MACHINE, rest
	}
	key, err := registry.OpenKey(root, name, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = key.Close() }()
	text, _, err := key.GetStringValue(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}
