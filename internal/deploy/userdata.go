package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"

	"opensteamtool-gui-mygo/internal/patch"
	"opensteamtool-gui-mygo/internal/source"
)

const recordName = "published.json"

func userDir(toolDir string, src source.Source) string {
	return filepath.Join(toolDir, "userdata", patch.Slug(src.Key))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func shippedHashesOrNone(host Files, dir string) map[string]string {
	path := filepath.Join(dir, recordName)
	if !host.IsFile(path) {
		return map[string]string{}
	}
	data, err := host.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	record := map[string]string{}
	if json.Unmarshal(data, &record) != nil {
		return map[string]string{}
	}
	return record
}

func storeRecord(host Files, dir string, record map[string]string) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return storeArchive(host, dir, recordName, data)
}

func storeArchive(host Files, dir string, name string, data []byte) error {
	if err := host.MkdirAll(dir); err != nil {
		return err
	}
	return writeAtomic(host, filepath.Join(dir, name), data)
}

func readArchive(host Files, dir string, name string) ([]byte, bool, error) {
	path := filepath.Join(dir, name)
	if !host.IsFile(path) {
		return nil, false, nil
	}
	data, err := host.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func deleteArchive(host Files, dir string, name string) error {
	path := filepath.Join(dir, name)
	if !host.IsFile(path) {
		return nil
	}
	return host.Remove(path)
}

func releaseUserFiles(host Files, root string, layout source.Layout, toolDir string, src source.Source) error {
	dir := userDir(toolDir, src)
	record := shippedHashesOrNone(host, dir)
	var unchanged []string
	for _, member := range layout.Manifest().UserOwned() {
		target := filepath.Join(root, member.Name)
		if !host.IsFile(target) {
			continue
		}
		data, err := host.ReadFile(target)
		if err != nil {
			return err
		}
		if record[member.Name] == sha256Hex(data) {
			unchanged = append(unchanged, member.Name)
			continue
		}
		if err := storeArchive(host, dir, member.Name, data); err != nil {
			return err
		}
	}
	for _, name := range unchanged {
		if err := deleteArchive(host, dir, name); err != nil {
			return err
		}
	}
	for _, member := range layout.Manifest().UserOwned() {
		target := filepath.Join(root, member.Name)
		if host.IsFile(target) {
			if err := host.Remove(target); err != nil {
				return err
			}
		}
		delete(record, member.Name)
	}
	if len(record) == 0 && !host.IsFile(filepath.Join(dir, recordName)) {
		return nil
	}
	return storeRecord(host, dir, record)
}
