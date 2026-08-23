//go:build !renvo && renvo_native_backend

package main

import (
	"bytes"
	"debug/elf"
	"os"
	"testing"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

func TestNativeBackendBuildCompilesInProcessWithoutPromotionCache(t *testing.T) {
	cacheDir := t.TempDir()
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc main() { print(\"PASS\\n\") }\n")},
	}
	compiled := driver.CompileUnit([]string{
		"-t", "linux/amd64", "-arena-size", "8192", "-s", "-o", "app", "./cmd/app",
	}, "/repo/case", "/std", files, checkoutBackend("/std", cacheDir))
	if !compiled.Ok {
		t.Fatalf("native backend compile failed: %#v", compiled.Diagnostic)
	}
	if len(compiled.Binary) < 4 || string(compiled.Binary[:4]) != "\x7fELF" {
		t.Fatalf("native backend output prefix = % x", compiled.Binary[:min(4, len(compiled.Binary))])
	}
	object := driver.CompileUnit([]string{
		"-t", "linux/amd64", "-mode=object", "-s", "-o", "app.o", "cmd/app/main.go",
	}, "/repo/case", "/std", files, checkoutBackend("/std", cacheDir))
	if !object.Ok {
		t.Fatalf("native backend object compile failed: %#v", object.Diagnostic)
	}
	parsedObject, err := elf.NewFile(bytes.NewReader(object.Binary))
	if err != nil {
		t.Fatalf("parse native backend object: %v", err)
	}
	if parsedObject.Type != elf.ET_REL || parsedObject.Machine != elf.EM_X86_64 {
		t.Fatalf("native backend object type/machine = %v/%v", parsedObject.Type, parsedObject.Machine)
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("in-process native compile populated promotion cache: %v", entries)
	}
}
