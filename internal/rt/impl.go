package rt

import "os"

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
