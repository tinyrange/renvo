//go:build !rtg

package os

import (
	stdfs "io/fs"
	stdos "os"
)

type FileMode = stdfs.FileMode

var Args = stdos.Args

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
	return stdos.WriteFile(name, data, perm)
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

func (f *File) Close() error {
	return f.file.Close()
}

func (d DirEntry) Name() string {
	return d.entry.Name()
}

func (d DirEntry) IsDir() bool {
	return d.entry.IsDir()
}
