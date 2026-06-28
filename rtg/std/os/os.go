//go:build !rtg

package os

const O_RDONLY = 0
const O_WRONLY = 1
const O_RDWR = 2
const O_CREATE = 64
const O_TRUNC = 512

const Stdin = 0
const Stdout = 1
const Stderr = 2

var Args []string

type FileInfo struct {
	name  string
	isDir bool
}

func (info FileInfo) IsDir() bool {
	return info.isDir
}

type DirEntry struct {
	name  string
	isDir bool
}

func (entry DirEntry) Name() string {
	return entry.name
}

func (entry DirEntry) IsDir() bool {
	return entry.isDir
}

func Open(path string, flags int) int {
	return -1
}

func Close(fd int) int {
	return -1
}

func Read(fd int, buf []byte, off int64) int {
	return -1
}

func Write(fd int, buf []byte, off int64) int {
	return -1
}

func Chmod(fd int, mode int) int {
	return -1
}

func Exit(code int) {
}

func Getenv(name string) string {
	return ""
}

func Getwd() (string, error) {
	return ".", nil
}

func Stat(path string) (FileInfo, error) {
	return FileInfo{}, statError("not implemented")
}

func IsNotExist(err error) bool {
	return err != nil
}

func ReadFile(path string) ([]byte, error) {
	return nil, statError("not implemented")
}

func WriteFile(path string, data []byte, mode int) error {
	return statError("not implemented")
}

func ReadDir(path string) ([]DirEntry, error) {
	return nil, statError("not implemented")
}

func MkdirAll(path string, mode int) error {
	return statError("not implemented")
}

type statError string

func (err statError) Error() string {
	return string(err)
}
