package rt

type Process struct {
	PID int
	Exe string
}

type Runtime interface {
	IsDir(path string) bool
	IsFile(path string) bool
	IsReadableDir(path string) bool
	Processes() ([]Process, error)
	Spawn(exe string, dir string, args ...string) (int, error)
	Terminate(pid int) error
	PickDirectory(title string, start string) (string, error)
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
	Unreadable map[string]bool
	Picked     string
	PickStart  string

	Procs       []Process
	Spawned     []Launch
	Killed      []int
	OnSpawn     func(Launch)
	OnTerminate func(int)
}
