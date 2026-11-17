package rt

type Process struct {
	PID int
	Exe string
}

type Runtime interface {
	IsDir(path string) bool
	IsFile(path string) bool
	Processes() ([]Process, error)
	Spawn(exe string, dir string, args ...string) (int, error)
	Terminate(pid int) error
}

type OS struct{}

type Launch struct {
	PID  int
	Exe  string
	Dir  string
	Args []string
}

type Fake struct {
	Dirs  map[string]bool
	Files map[string]bool

	Procs       []Process
	Spawned     []Launch
	Killed      []int
	OnSpawn     func(Launch)
	OnTerminate func(int)
}
