package rt

import (
	"os"
	"slices"
)

func (OS) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (OS) IsFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (f Fake) IsDir(path string) bool { return f.Dirs[path] }

func (f Fake) IsFile(path string) bool { return f.Files[path] }

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
