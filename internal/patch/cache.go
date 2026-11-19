package patch

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"opensteamtool-gui-mygo/internal/source"
)

const VersionFile = "version.txt"

type State struct {
	Version  string
	Present  bool
	Complete bool
}

func Dir(toolDir string, src source.Source) string {
	return filepath.Join(toolDir, "patches", Slug(src.Key))
}

func Slug(key string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(key, "https://"), "github.com/")
	clean := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.':
			return r
		}
		return '-'
	}, trimmed)
	return strings.Trim(clean, "-")
}

func Install(dir string, files []File, version string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("make the cache directory: %w", err)
	}
	for _, file := range files {
		if err := write(filepath.Join(dir, file.Name), file.Data); err != nil {
			return err
		}
	}
	if err := write(filepath.Join(dir, VersionFile), []byte(version+"\n")); err != nil {
		return err
	}
	return prune(dir, files)
}

func prune(dir string, files []File) error {
	held, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read the cache directory: %w", err)
	}
	wanted := map[string]bool{VersionFile: true}
	for _, file := range files {
		wanted[file.Name] = true
	}
	for _, entry := range held {
		if entry.IsDir() || wanted[entry.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("clear %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func write(target string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(target), filepath.Base(target)+".*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	name := temp.Name()
	discard := func() {
		_ = temp.Close()
		_ = os.Remove(name)
	}
	if _, err := temp.Write(data); err != nil {
		discard()
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("write %s: %w", target, err)
	}
	if err := os.Rename(name, target); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("write %s: %w", target, err)
	}
	return nil
}

func Inspect(dir string, layout source.Layout, entry source.Entry) State {
	state := State{}
	data, err := os.ReadFile(filepath.Join(dir, VersionFile))
	if err == nil {
		state.Version = strings.TrimSpace(string(data))
		state.Present = state.Version != ""
	}
	manifest := layout.Manifest()
	held := true
	for _, member := range manifest.Required(entry) {
		if info, err := os.Stat(filepath.Join(dir, member.Name)); err != nil || info.IsDir() {
			held = false
		}
	}
	state.Complete = held
	return state
}
func Load(dir string, layout source.Layout, entry source.Entry) ([]File, error) {
	manifest := layout.Manifest()
	required := manifest.Required(entry)
	files := make([]File, 0, len(required)+len(manifest.Optional))
	for _, member := range required {
		data, err := os.ReadFile(filepath.Join(dir, member.Name))
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrMissingMember, member.Name)
		}
		files = append(files, File{Name: member.Name, Data: data})
	}
	for _, member := range manifest.Optional {
		data, err := os.ReadFile(filepath.Join(dir, member.Name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", member.Name, err)
		}
		files = append(files, File{Name: member.Name, Data: data})
	}
	return files, nil
}

var clientBuild = regexp.MustCompile(`(?m)^\s*"version"\s+"([^"]*)"`)

func ClientBuild(steamDir string) string {
	data, err := os.ReadFile(filepath.Join(steamDir, "package", "steam_client_win32.manifest"))
	if err != nil {
		return ""
	}
	found := clientBuild.FindSubmatch(data)
	if found == nil {
		return ""
	}
	return string(found[1])
}
