package patch

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnparsed      = errors.New("the release cannot be parsed")
	ErrNoAsset       = errors.New("the release carries no archive to download")
	ErrArchivesOnly  = errors.New("the release carries only non-zip archives")
	ErrBroken        = errors.New("the archive cannot be read")
	ErrMissingMember = errors.New("the archive is missing a required member")
)

type Asset struct {
	Name string
	URL  string
}

type Release struct {
	Version string
	Assets  []Asset
}

func LatestURL(repository string) string {
	return "https://api.github.com/repos/" + repository + "/releases/latest"
}

func Parse(data []byte) (Release, error) {
	var body struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return Release{}, fmt.Errorf("%w: %v", ErrUnparsed, err)
	}
	version := strings.TrimSpace(body.TagName)
	if version == "" {
		return Release{}, fmt.Errorf("%w: no tag_name", ErrUnparsed)
	}
	release := Release{Version: version, Assets: make([]Asset, 0, len(body.Assets))}
	for _, asset := range body.Assets {
		release.Assets = append(release.Assets, Asset{Name: asset.Name, URL: asset.URL})
	}
	return release, nil
}

var archiveSuffixes = []string{".7z", ".rar", ".tar.gz", ".tgz", ".tar", ".gz", ".xz", ".bz2"}

func Pick(assets []Asset) (Asset, error) {
	for _, asset := range assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), "-release.zip") {
			return asset, nil
		}
	}
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, "-debug.zip") {
			return asset, nil
		}
	}
	for _, asset := range assets {
		if isArchive(asset.Name) {
			return Asset{}, ErrArchivesOnly
		}
	}
	return Asset{}, ErrNoAsset
}

func isArchive(name string) bool {
	name = strings.ToLower(name)
	for _, suffix := range archiveSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
