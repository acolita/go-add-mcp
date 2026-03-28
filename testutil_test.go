package addmcp

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// --- memFS: in-memory filesystem for testing ---

type memFS struct {
	files map[string][]byte
	dirs  map[string]bool
}

func newMemFS() *memFS {
	return &memFS{files: map[string][]byte{}, dirs: map[string]bool{}}
}

func (m *memFS) ReadFile(path string) ([]byte, error) {
	if data, ok := m.files[path]; ok {
		return slices.Clone(data), nil
	}
	return nil, &os.PathError{Op: "read", Path: path, Err: fs.ErrNotExist}
}

func (m *memFS) WriteFile(path string, data []byte, _ os.FileMode) error {
	m.files[path] = slices.Clone(data)
	return nil
}

func (m *memFS) MkdirAll(path string, _ os.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *memFS) Stat(path string) (os.FileInfo, error) {
	if m.dirs[path] {
		return memFileInfo{name: filepath.Base(path), dir: true}, nil
	}
	if data, ok := m.files[path]; ok {
		return memFileInfo{name: filepath.Base(path), size: int64(len(data))}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: path, Err: fs.ErrNotExist}
}

func (m *memFS) Remove(path string) error {
	if _, ok := m.files[path]; ok {
		delete(m.files, path)
		return nil
	}
	return &os.PathError{Op: "remove", Path: path, Err: fs.ErrNotExist}
}

// putJSON stores pre-existing JSON content for a config file.
func (m *memFS) putJSON(path string, data []byte) {
	m.files[path] = slices.Clone(data)
}

type memFileInfo struct {
	name string
	size int64
	dir  bool
}

func (f memFileInfo) Name() string    { return f.name }
func (f memFileInfo) Size() int64     { return f.size }
func (f memFileInfo) IsDir() bool     { return f.dir }
func (f memFileInfo) Sys() any        { return nil }
func (f memFileInfo) ModTime() time.Time { return time.Time{} }
func (f memFileInfo) Mode() os.FileMode {
	if f.dir {
		return os.ModeDir | 0755
	}
	return 0644
}

// --- fakeDetector ---

type fakeDetector struct {
	dirs     map[string]bool
	commands map[string]bool
}

func (f fakeDetector) DirExists(path string) bool    { return f.dirs[path] }
func (f fakeDetector) CommandExists(name string) bool { return f.commands[name] }
