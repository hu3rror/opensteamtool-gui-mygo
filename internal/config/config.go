package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"opensteamtool-gui-mygo/internal/app"
	"opensteamtool-gui-mygo/internal/rt"
)

const (
	FileName = "config.json"
	Version  = 1
)

type Config struct {
	SteamPath string
	Language  app.Language
}

type wire struct {
	Version   int     `json:"version"`
	SteamPath *string `json:"steam_path"`
	Language  *string `json:"language"`
}

var languageNames = map[app.Language]string{
	app.LanguageAuto:    "auto",
	app.LanguageChinese: "zh",
	app.LanguageEnglish: "en",
}

func Load(dir string, host rt.Runtime) (Config, bool, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, fmt.Errorf("read config: %w", err)
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return Config{}, false, fmt.Errorf("read config: %w", err)
	}
	if w.Version != Version {
		return Config{}, false, fmt.Errorf("read config: version %d is not %d", w.Version, Version)
	}
	if w.SteamPath == nil {
		return Config{}, false, errors.New("read config: the Steam path is missing")
	}
	if w.Language == nil {
		return Config{}, false, errors.New("read config: the language is missing")
	}
	language, ok := languageFrom(*w.Language)
	if !ok {
		return Config{}, false, fmt.Errorf("read config: %q is not a language", *w.Language)
	}
	cfg := Config{SteamPath: *w.SteamPath, Language: language}
	if cfg.SteamPath != "" && (!host.IsDir(cfg.SteamPath) || !host.IsReadableDir(cfg.SteamPath)) {
		return cfg, false, fmt.Errorf("read config: %q is not a readable directory", cfg.SteamPath)
	}
	return cfg, true, nil
}

func Save(dir string, cfg Config) error {
	language, ok := languageNames[cfg.Language]
	if !ok {
		return fmt.Errorf("save config: %d is not a language", cfg.Language)
	}
	data, err := json.MarshalIndent(wire{Version: Version, SteamPath: &cfg.SteamPath, Language: &language}, "", "  ")
	if err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if err := replace(dir, append(data, '\n')); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

func languageFrom(name string) (app.Language, bool) {
	for language, candidate := range languageNames {
		if candidate == name {
			return language, true
		}
	}
	return app.LanguageAuto, false
}

func replace(dir string, data []byte) error {
	temp, err := os.CreateTemp(dir, FileName+".*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	discard := func() {
		_ = temp.Close()
		_ = os.Remove(name)
	}
	if _, err := temp.Write(data); err != nil {
		discard()
		return err
	}
	if err := temp.Sync(); err != nil {
		discard()
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, filepath.Join(dir, FileName)); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
