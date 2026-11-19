package rt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/egoist/mygo"
)

func (OS) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (OS) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (OS) ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

func (OS) Stat(path string) (Stat, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Stat{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return Stat{Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (OS) IsReadableDir(path string) bool {
	dir, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = dir.Close() }()
	_, err = dir.Readdirnames(1)
	return err == nil || errors.Is(err, io.EOF)
}

func (OS) MkdirAll(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("make %s: %w", path, err)
	}
	return nil
}

func (OS) WriteFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func (OS) Rename(from string, to string) error {
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("rename %s to %s: %w", from, to, err)
	}
	return nil
}

func (OS) Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

func (OS) Get(url string) ([]byte, error) {
	response, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: %s", url, response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	return body, nil
}

func (OS) Fetch(url string, timeout time.Duration) (Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Response{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return Response{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Response{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	return Response{Status: response.StatusCode, Body: body}, nil
}

func (OS) PickDirectory(title string, start string) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: title, DefaultPath: start, Directory: true})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

func (OS) OpenURL(url string) error {
	if err := mygo.Shell.OpenExternal(url); err != nil {
		return fmt.Errorf("open %s: %w", url, err)
	}
	return nil
}

func (OS) OpenPath(path string) error {
	if err := mygo.Shell.OpenPath(path); err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	return nil
}

func (f Fake) IsDir(path string) bool { return f.Dirs[path] }

func (f Fake) IsReadableDir(path string) bool { return f.Dirs[path] && !f.Unreadable[path] }

func (f *Fake) PickDirectory(_ string, start string) (string, error) {
	f.PickStart = start
	return f.Picked, nil
}
func (f *Fake) OpenPath(path string) error {
	f.Opens = append(f.Opens, path)
	return f.OpenError
}

func (f *Fake) OpenURL(url string) error {
	f.OpenedURLs = append(f.OpenedURLs, url)
	return f.OpenURLError
}

func (f Fake) IsFile(path string) bool { return f.Files[path] }

func (f *Fake) Stat(path string) (Stat, error) {
	if !f.Files[path] {
		return Stat{}, fmt.Errorf("stat %s: the fake has no such file", path)
	}
	return Stat{Size: int64(len(f.Contents[path])), ModTime: f.ModTimes[path]}, nil
}

func (f *Fake) ReadFile(path string) ([]byte, error) {
	f.Reads = append(f.Reads, path)
	if !f.Files[path] {
		return nil, fmt.Errorf("read %s: the fake has no such file", path)
	}
	data, ok := f.Contents[path]
	if !ok {
		return nil, fmt.Errorf("read %s: the fake holds no content for the file", path)
	}
	return slices.Clone(data), nil
}

func (f *Fake) MkdirAll(path string) error {
	f.Mkdirs = append(f.Mkdirs, path)
	if f.Dirs == nil {
		f.Dirs = map[string]bool{}
	}
	f.Dirs[path] = true
	return nil
}

func (f *Fake) WriteFile(path string, data []byte) error {
	f.Writes = append(f.Writes, Write{Path: path, Data: data})
	if f.Files == nil {
		f.Files = map[string]bool{}
	}
	f.Files[path] = true
	if f.Contents == nil {
		f.Contents = map[string][]byte{}
	}
	f.Contents[path] = slices.Clone(data)
	if f.ModTimes == nil {
		f.ModTimes = map[string]time.Time{}
	}
	f.clock = f.clock.Add(time.Second)
	f.ModTimes[path] = f.clock
	return nil
}

func (f *Fake) Rename(from string, to string) error {
	f.Renames = append(f.Renames, Move{From: from, To: to})
	delete(f.Files, from)
	if f.Files == nil {
		f.Files = map[string]bool{}
	}
	f.Files[to] = true
	if data, ok := f.Contents[from]; ok {
		f.Contents[to] = data
		delete(f.Contents, from)
	}
	if modTime, ok := f.ModTimes[from]; ok {
		f.ModTimes[to] = modTime
		delete(f.ModTimes, from)
	}
	return nil
}

func (f *Fake) Remove(path string) error {
	f.Removes = append(f.Removes, path)
	delete(f.Files, path)
	delete(f.Contents, path)
	delete(f.ModTimes, path)
	return nil
}

func (f *Fake) Get(url string) ([]byte, error) {
	f.Fetched = append(f.Fetched, url)
	if f.FetchError != nil {
		return nil, f.FetchError
	}
	body, ok := f.Responses[url]
	if !ok {
		return nil, fmt.Errorf("get %s: the fake has no response", url)
	}
	return body, nil
}

func (f *Fake) Fetch(url string, timeout time.Duration) (Response, error) {
	f.Fetched = append(f.Fetched, url)
	if f.OnFetch != nil {
		f.OnFetch(url)
	}
	answer, ok := f.Answers[url]
	if !ok {
		return Response{}, fmt.Errorf("fetch %s: the fake has no answer", url)
	}
	if answer.Err != nil {
		return Response{}, answer.Err
	}
	status := answer.Status
	if status == 0 {
		status = 200
	}
	return Response{Status: status, Body: slices.Clone(answer.Body)}, nil
}

func (f *Fake) Processes() ([]Process, error) { return slices.Clone(f.Procs), nil }

func (f *Fake) Spawn(exe string, dir string, args ...string) (int, error) {
	call := Launch{PID: 1000 + len(f.Spawned), Exe: exe, Dir: dir, Args: args}
	f.Spawned = append(f.Spawned, call)
	if f.OnSpawn != nil {
		f.OnSpawn(call)
	}
	return call.PID, nil
}

func (f *Fake) Terminate(pid int) error {
	f.Killed = append(f.Killed, pid)
	if f.OnTerminate != nil {
		f.OnTerminate(pid)
		return nil
	}
	f.Procs = slices.DeleteFunc(f.Procs, func(p Process) bool { return p.PID == pid })
	return nil
}
