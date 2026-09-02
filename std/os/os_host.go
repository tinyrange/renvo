//go:build !renvo

package os

import (
	stdfs "io/fs"
	stdos "os"
)

const (
	O_RDONLY = stdos.O_RDONLY
	O_WRONLY = stdos.O_WRONLY
	O_RDWR   = stdos.O_RDWR
	O_CREATE = stdos.O_CREATE
	O_TRUNC  = stdos.O_TRUNC
)

var Args = stdos.Args
var Stdin = &File{file: stdos.Stdin}
var Stdout = &File{file: stdos.Stdout}
var Stderr = &File{file: stdos.Stderr}

type File struct {
	file *stdos.File
}

type DirEntry struct {
	entry stdos.DirEntry
}

func Environ() []string {
	return stdos.Environ()
}

func Exit(code int) {
	stdos.Exit(code)
}

func Getwd() (string, error) {
	return stdos.Getwd()
}

func ReadFile(name string) ([]byte, error) {
	return stdos.ReadFile(name)
}

func WriteFile(name string, data []byte, perm FileMode) error {
	return stdos.WriteFile(name, data, stdfs.FileMode(perm))
}

func ReadDir(name string) ([]DirEntry, error) {
	entries, err := stdos.ReadDir(name)
	if err != nil {
		return nil, err
	}
	out := make([]DirEntry, 0, len(entries))
	for i := 0; i < len(entries); i++ {
		out = append(out, DirEntry{entry: entries[i]})
	}
	return out, nil
}

func Open(name string) (*File, error) {
	file, err := stdos.Open(name)
	if err != nil {
		return nil, err
	}
	return &File{file: file}, nil
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	file, err := stdos.OpenFile(name, flag, stdfs.FileMode(perm))
	if err != nil {
		return nil, err
	}
	return &File{file: file}, nil
}

func Create(name string) (*File, error) {
	file, err := stdos.Create(name)
	if err != nil {
		return nil, err
	}
	return &File{file: file}, nil
}

func (f *File) Read(p []byte) (int, error) {
	return f.file.Read(p)
}

func (f *File) Write(p []byte) (int, error) {
	return f.file.Write(p)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.file.Seek(offset, whence)
}

func (f *File) Close() error {
	return f.file.Close()
}

func (d DirEntry) Name() string {
	return d.entry.Name()
}

func (d DirEntry) IsDir() bool {
	return d.entry.IsDir()
}
