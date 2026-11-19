package deploy

import (
	"fmt"
	"path/filepath"

	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
)

type Files interface {
	IsFile(path string) bool
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string) error
	WriteFile(path string, data []byte) error
	Rename(from string, to string) error
	Remove(path string) error
}

type Plan struct {
	Root    string
	ToolDir string
	Layout  source.Layout
	Entry   source.Entry
	Source  source.Source
	Files   []patch.File
	Remove  string
}

type Removal struct {
	Root    string
	Layout  source.Layout
	ToolDir string
	Source  source.Source
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

func Uninstall(host Files, removal Removal) error {
	if len(removal.Layout.Manifest().UserOwned()) > 0 {
		if err := releaseUserFiles(host, removal.Root, removal.Layout, removal.ToolDir, removal.Source); err != nil {
			return err
		}
	}
	for _, name := range removal.Layout.Manifest().RootFiles() {
		target := filepath.Join(removal.Root, name)
		if !host.IsFile(target) {
			continue
		}
		if err := host.Remove(target); err != nil {
			return err
		}
	}
	return nil
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
	other := plan.Layout.Other()
	if len(other.Manifest().UserOwned()) > 0 {
		owner, ok := source.BuiltinFor(other)
		if !ok {
			return fmt.Errorf("no single built-in source locks the other layout")
		}
		if err := releaseUserFiles(host, plan.Root, other, plan.ToolDir, owner); err != nil {
			return err
		}
	}
	own := map[string]bool{}
	for _, name := range plan.Layout.Manifest().RootFiles() {
		own[name] = true
	}
	for _, name := range other.Manifest().RootFiles() {
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

func writeAtomic(host Files, target string, data []byte) error {
	temp := target + ".tmp"
	if err := host.WriteFile(temp, data); err != nil {
		return err
	}
	if err := host.Rename(temp, target); err != nil {
		_ = host.Remove(temp)
		return err
	}
	return nil
}

func writeAll(host Files, plan Plan) error {
	owned := map[string]bool{}
	for _, member := range plan.Layout.Manifest().UserOwned() {
		owned[member.Name] = true
	}
	dir := userDir(plan.ToolDir, plan.Source)
	record := map[string]string{}
	if len(owned) > 0 {
		record = shippedHashesOrNone(host, dir)
	}
	recorded := false
	for _, file := range plan.Files {
		target := filepath.Join(plan.Root, file.Name)
		data := file.Data
		if owned[file.Name] {
			if host.IsFile(target) {
				continue
			}
			restored, ok, err := readArchive(host, dir, file.Name)
			if err != nil {
				return err
			}
			if ok {
				data = restored
			}
			record[file.Name] = sha256Hex(file.Data)
			recorded = true
		}
		if err := writeAtomic(host, target, data); err != nil {
			return err
		}
	}
	if !recorded {
		return nil
	}
	return storeRecord(host, dir, record)
}
