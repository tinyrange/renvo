package driver

import (
	"bytes"
	"debug/pe"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompileCommandVirtualBackend(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{"windows_aarch64.rtg", "aarch64.rtg"} {
		data, err := os.ReadFile("../backend/definitions/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if name == "windows_aarch64.rtg" {
			data = bytes.ReplaceAll(data, []byte("target windows/arm64"), []byte("target custom/arm64"))
		}
		files["definitions/"+name] = &fstest.MapFile{Data: data}
	}
	files["main.go"] = &fstest.MapFile{Data: []byte("package main\nfunc appMain() int { return 37 }\n")}
	fs := memorySourceFS{files: files}
	r, err := CompileCommand(&CommandRequest{Filesystem: fs,
		Args:   []string{"-backend", "definitions/windows_aarch64.rtg", "-o", "app.exe", "main.go"},
		Target: "custom/arm64", ArenaSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok {
		t.Fatalf("compile: %+v", r.Diagnostic)
	}
	image, err := pe.NewFile(bytes.NewReader(r.Binary))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	header, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || image.Machine != pe.IMAGE_FILE_MACHINE_ARM64 || header.AddressOfEntryPoint == 0 {
		t.Fatalf("invalid ARM64 image: %+v", image.FileHeader)
	}
	if !bytes.Equal(r.Outputs["app.exe"], r.Binary) {
		t.Fatal("missing virtual output")
	}
	delete(files, "definitions/aarch64.rtg")
	r, err = CompileCommand(&CommandRequest{Filesystem: fs,
		Args: []string{"-backend", "definitions/windows_aarch64.rtg", "main.go"}, Target: "custom/arm64"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Ok || !strings.Contains(r.Diagnostic.Code, "RTG") {
		t.Fatalf("missing virtual import: %+v", r)
	}
}

func TestCompileCommandKernelPackage(t *testing.T) {
	files := fstest.MapFS{}
	for _, name := range []string{"drivers/windows_kernel_aarch64.rtg", "aarch64.rtg"} {
		data, err := os.ReadFile("../backend/definitions/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files["definitions/"+name] = &fstest.MapFile{Data: data}
	}
	files["go.mod"] = &fstest.MapFile{Data: []byte("module driverprobe\n")}
	files["main.go"] = &fstest.MapFile{Data: []byte("package main\n//export DriverEntry\nfunc DriverEntry(driver, path uintptr) int32 { return helper(driver, path) }\n")}
	files["helper.go"] = &fstest.MapFile{Data: []byte("package main\nfunc helper(driver,path uintptr) int32 { if driver==0 || path==0 { return -1 }; return 0 }\n")}
	for _, input := range [][]string{{"."}, {"main.go", "helper.go"}} {
		args := []string{"-backend", "definitions/drivers/windows_kernel_aarch64.rtg", "-mode=object", "-o", "probe.sys"}
		args = append(args, input...)
		r, err := CompileCommand(&CommandRequest{Filesystem: memorySourceFS{files: files}, Args: args, Target: "windows-kernel/arm64", ArenaSize: 1 << 20})
		if err != nil {
			t.Fatal(err)
		}
		if !r.Ok {
			t.Fatalf("%v: %+v", input, r.Diagnostic)
		}
		image, err := pe.NewFile(bytes.NewReader(r.Binary))
		if err != nil {
			t.Fatal(err)
		}
		header := image.OptionalHeader.(*pe.OptionalHeader64)
		if image.Machine != pe.IMAGE_FILE_MACHINE_ARM64 || header.Subsystem != 1 || header.ImageBase != 0x140000000 {
			t.Fatalf("invalid kernel header: %+v", header)
		}
		image.Close()
	}
}
