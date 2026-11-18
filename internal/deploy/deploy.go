package deploy

import (
	"path/filepath"

	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
)

type Files interface {
	IsFile(path string) bool
	MkdirAll(path string) error
	WriteFile(path string, data []byte) error
	Rename(from string, to string) error
	Remove(path string) error
}

type Plan struct {
	Root   string
	Layout source.Layout
	Entry  source.Entry
	Files  []patch.File
	Remove string
}

type Conflict struct {
	Name string
}

func (c Conflict) Error() string {
	return "the file " + c.Name + " is in the way of the patch"
}

func Write(host Files, plan Plan) error {
	if err := resolveConflict(host, plan); err != nil {
		return err
	}
	if err := host.MkdirAll(filepath.Join(plan.Root, filepath.FromSlash(plan.Layout.Manifest().Lua))); err != nil {
		return err
	}
	if err := clearOther(host, plan); err != nil {
		return err
	}
	return writeAll(host, plan)
}

func conflicting(host Files, layout source.Layout, entry source.Entry, root string) (string, bool) {
	manifest := layout.Manifest()
	for _, member := range manifest.Entries {
		if member.Entry == entry {
			continue
		}
		if host.IsFile(filepath.Join(root, member.Name)) {
			return member.Name, true
		}
	}
	for _, name := range manifest.Forbidden {
		if host.IsFile(filepath.Join(root, name)) {
			return name, true
		}
	}
	return "", false
}

func resolveConflict(host Files, plan Plan) error {
	name, found := conflicting(host, plan.Layout, plan.Entry, plan.Root)
	if !found {
		return nil
	}
	if plan.Remove != name {
		return Conflict{Name: name}
	}
	return host.Remove(filepath.Join(plan.Root, name))
}

func clearOther(host Files, plan Plan) error {
	own := map[string]bool{}
	for _, name := range plan.Layout.Manifest().RootFiles() {
		own[name] = true
	}
	for _, name := range plan.Layout.Other().Manifest().RootFiles() {
		target := filepath.Join(plan.Root, name)
		if own[name] || !host.IsFile(target) {
			continue
		}
		if err := host.Remove(target); err != nil {
			return err
		}
	}
	return nil
}

func writeAll(host Files, plan Plan) error {
	keep := map[string]bool{}
	for _, member := range plan.Layout.Manifest().Optional {
		if member.Keep {
			keep[member.Name] = true
		}
	}
	for _, file := range plan.Files {
		target := filepath.Join(plan.Root, file.Name)
		if keep[file.Name] && host.IsFile(target) {
			continue
		}
		temp := target + ".tmp"
		if err := host.WriteFile(temp, file.Data); err != nil {
			return err
		}
		if err := host.Rename(temp, target); err != nil {
			_ = host.Remove(temp)
			return err
		}
	}
	return nil
}
