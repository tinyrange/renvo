package os

import (
	hostos "os"
	"path/filepath"
	"testing"
)

func TestFileOperations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	data, err := ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("ReadFile = %q, %v", string(data), err)
	}
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	buf := make([]byte, 2)
	n, err := file.Read(buf)
	if err != nil || n != 2 || string(buf) != "he" {
		t.Fatalf("Read = %d, %v, %q", n, err, string(buf))
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestFileSeek(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seek.txt")
	file, err := OpenFile(path, O_CREATE|O_TRUNC|O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err = file.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if offset, seekErr := file.Seek(-2, 1); seekErr != nil || offset != 4 {
		t.Fatalf("Seek current = %d, %v", offset, seekErr)
	}
	if _, err = file.Write([]byte("XY")); err != nil {
		t.Fatal(err)
	}
	if offset, seekErr := file.Seek(-2, 2); seekErr != nil || offset != 4 {
		t.Fatalf("Seek end = %d, %v", offset, seekErr)
	}
	buf := make([]byte, 2)
	if n, readErr := file.Read(buf); readErr != nil || n != 2 || string(buf) != "XY" {
		t.Fatalf("Read after Seek = %d, %v, %q", n, readErr, buf)
	}
}

func TestReadDirAndEnv(t *testing.T) {
	dir := t.TempDir()
	if err := hostos.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := hostos.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadDir(dir)
	if err != nil || len(entries) != 2 || entries[0].Name() != "a.txt" || entries[1].IsDir() {
		t.Fatalf("ReadDir = %#v, %v", entries, err)
	}
	if wd, err := Getwd(); err != nil || wd == "" {
		t.Fatalf("Getwd = %q, %v", wd, err)
	}
	if len(Environ()) == 0 {
		t.Fatalf("Environ returned empty environment")
	}
}
