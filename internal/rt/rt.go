package rt

type Runtime interface {
	IsDir(path string) bool
	IsFile(path string) bool
}

type OS struct{}

type Fake struct {
	Dirs  map[string]bool
	Files map[string]bool
}
