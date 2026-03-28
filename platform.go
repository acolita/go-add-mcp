package addmcp

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
)

// Platform captures environment-dependent values, enabling pure path resolution
// and detection logic that can be tested across platforms from any OS.
type Platform struct {
	GOOS       string // runtime.GOOS
	HomeDir    string // os.UserHomeDir()
	AppData    string // %APPDATA% on Windows, empty elsewhere
	WorkingDir string // os.Getwd() fallback for project scope
}

// DefaultPlatform reads the real environment.
func DefaultPlatform() Platform {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	return Platform{
		GOOS:       runtime.GOOS,
		HomeDir:    home,
		AppData:    os.Getenv("APPDATA"),
		WorkingDir: wd,
	}
}

// FS abstracts filesystem operations for testability.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Stat(path string) (os.FileInfo, error)
	Remove(path string) error
}

type osFS struct{}

func (osFS) ReadFile(path string) ([]byte, error)                 { return os.ReadFile(path) }
func (osFS) WriteFile(path string, d []byte, p os.FileMode) error { return os.WriteFile(path, d, p) }
func (osFS) MkdirAll(path string, p os.FileMode) error            { return os.MkdirAll(path, p) }
func (osFS) Stat(path string) (os.FileInfo, error)                { return os.Stat(path) }
func (osFS) Remove(path string) error                             { return os.Remove(path) }

// Detector abstracts checks for installed software.
type Detector interface {
	DirExists(path string) bool
	CommandExists(name string) bool
}

type realDetector struct{}

func (realDetector) DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (realDetector) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
