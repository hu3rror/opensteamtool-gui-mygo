package patch

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"opensteamtool-gui-mygo/internal/source"
)

type File struct {
	Name string
	Data []byte
}

func Extract(archive []byte, layout source.Layout, entry source.Entry) ([]File, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBroken, err)
	}
	published := map[string][]byte{}
	for _, part := range reader.File {
		if part.FileInfo().IsDir() {
			continue
		}
		key := strings.ToLower(path.Base(strings.ReplaceAll(part.Name, `\`, "/")))
		if _, seen := published[key]; seen {
			continue
		}
		data, err := read(part)
		if err != nil {
			return nil, err
		}
		published[key] = data
	}

	manifest := layout.Manifest()
	required := manifest.Required(entry)
	files := make([]File, 0, len(required)+len(manifest.Optional))
	for _, member := range required {
		data, ok := lookup(published, member)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingMember, member.Name)
		}
		files = append(files, File{Name: member.Name, Data: data})
	}
	for _, member := range manifest.Optional {
		if data, ok := lookup(published, member); ok {
			files = append(files, File{Name: member.Name, Data: data})
		}
	}
	return files, nil
}

func lookup(published map[string][]byte, member source.Member) ([]byte, bool) {
	for _, name := range member.Published {
		if data, ok := published[strings.ToLower(name)]; ok {
			return data, true
		}
	}
	return nil, false
}

func read(part *zip.File) ([]byte, error) {
	handle, err := part.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBroken, err)
	}
	defer func() { _ = handle.Close() }()
	data, err := io.ReadAll(handle)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBroken, err)
	}
	return data, nil
}
