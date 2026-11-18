package rt

import (
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

func (OS) IsReadableDir(path string) bool {
	dir, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = dir.Close() }()
	_, err = dir.Readdirnames(1)
	return err == nil || errors.Is(err, io.EOF)
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

func (OS) PickDirectory(title string, start string) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Title: title, DefaultPath: start, Directory: true})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

func (f Fake) IsDir(path string) bool { return f.Dirs[path] }

func (f Fake) IsReadableDir(path string) bool { return f.Dirs[path] && !f.Unreadable[path] }

func (f *Fake) PickDirectory(_ string, start string) (string, error) {
	f.PickStart = start
	return f.Picked, nil
}

func (f Fake) IsFile(path string) bool { return f.Files[path] }

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
