package rt

import "time"

type Process struct {
	PID int
	Exe string
}

type Response struct {
	Status int
	Body   []byte
}

type Answer struct {
	Status int
	Body   []byte
	Err    error
}

type Runtime interface {
	IsDir(path string) bool
	IsFile(path string) bool
	ReadFile(path string) ([]byte, error)
	IsReadableDir(path string) bool
	MkdirAll(path string) error
	WriteFile(path string, data []byte) error
	Rename(from string, to string) error
	Remove(path string) error
	Processes() ([]Process, error)
	Spawn(exe string, dir string, args ...string) (int, error)
	Terminate(pid int) error
	PickDirectory(title string, start string) (string, error)
	OpenPath(path string) error
	Get(url string) ([]byte, error)
	Fetch(url string, timeout time.Duration) (Response, error)
}

type OS struct{}

type Launch struct {
	PID  int
	Exe  string
	Dir  string
	Args []string
}

type Fake struct {
	Dirs       map[string]bool
	Files      map[string]bool
	Contents   map[string][]byte
	Unreadable map[string]bool
	Picked     string
	PickStart  string

	Opens     []string
	OpenError error

	Responses  map[string][]byte
	FetchError error
	Answers    map[string]Answer
	Fetched    []string

	Mkdirs  []string
	Writes  []Write
	Renames []Move
	Removes []string

	Procs       []Process
	Spawned     []Launch
	Killed      []int
	OnSpawn     func(Launch)
	OnTerminate func(int)
}

type Write struct {
	Path string
	Data []byte
}

type Move struct {
	From string
	To   string
}
