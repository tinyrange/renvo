//go:build !renvo

package backendcompiled

import (
	"io"
	"os"
	"sync"
)

const (
	O_RDWR   = os.O_RDWR
	O_RDONLY = os.O_RDONLY
	O_WRONLY = os.O_WRONLY
	O_CREATE = os.O_CREATE
	O_TRUNC  = os.O_TRUNC
)

var hostFiles = struct {
	sync.Mutex
	next  int
	files map[int]*os.File
}{next: 3, files: make(map[int]*os.File)}

func open(path string, flags int) int {
	file, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		return -1
	}
	hostFiles.Lock()
	fd := hostFiles.next
	hostFiles.next++
	hostFiles.files[fd] = file
	hostFiles.Unlock()
	return fd
}

func close(fd int) int {
	hostFiles.Lock()
	file := hostFiles.files[fd]
	delete(hostFiles.files, fd)
	hostFiles.Unlock()
	if file == nil || file.Close() != nil {
		return -1
	}
	return 0
}

func read(fd int, buffer []byte, offset int64) int {
	file := hostFile(fd)
	if file == nil {
		return -1
	}
	var n int
	var err error
	if offset < 0 {
		n, err = file.Read(buffer)
	} else {
		n, err = file.ReadAt(buffer, offset)
	}
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return -1
	}
	return n
}

func write(fd int, buffer []byte, offset int64) int {
	file := hostFile(fd)
	if file == nil {
		return -1
	}
	var n int
	var err error
	if offset < 0 {
		n, err = file.Write(buffer)
	} else {
		n, err = file.WriteAt(buffer, offset)
	}
	if err != nil {
		return -1
	}
	return n
}

func chmod(fd int, mode int) int {
	file := hostFile(fd)
	if file == nil || file.Chmod(os.FileMode(mode)) != nil {
		return -1
	}
	return 0
}

func print(value string) {
	_, _ = os.Stdout.WriteString(value)
}

func hostFile(fd int) *os.File {
	if fd == 0 {
		return os.Stdin
	}
	if fd == 1 {
		return os.Stdout
	}
	if fd == 2 {
		return os.Stderr
	}
	hostFiles.Lock()
	file := hostFiles.files[fd]
	hostFiles.Unlock()
	return file
}
