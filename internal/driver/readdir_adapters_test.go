package driver

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDirAdapterSelection(t *testing.T) {
	osDir := filepath.Join("..", "..", "std", "os")
	adapters := []string{
		"readdir_linux_amd64_renvo.go",
		"readdir_linux_386_renvo.go",
		"readdir_linux_aarch64_renvo.go",
		"readdir_linux_arm_renvo.go",
		"readdir_darwin_arm64_renvo.go",
		"readdir_wasi_renvo.go",
		"readdir_windows_renvo.go",
	}
	tests := []struct {
		goos string
		arch string
		want string
	}{
		{"linux", "amd64", "readdir_linux_amd64_renvo.go"},
		{"linux", "386", "readdir_linux_386_renvo.go"},
		{"linux", "arm64", "readdir_linux_aarch64_renvo.go"},
		{"linux", "arm", "readdir_linux_arm_renvo.go"},
		{"darwin", "arm64", "readdir_darwin_arm64_renvo.go"},
		{"wasip1", "wasm", "readdir_wasi_renvo.go"},
		{"windows", "amd64", "readdir_windows_renvo.go"},
	}
	for _, test := range tests {
		context := build.Default
		context.GOOS = test.goos
		context.GOARCH = test.arch
		context.BuildTags = []string{"renvo"}
		var selected []string
		for _, name := range adapters {
			matched, err := context.MatchFile(osDir, name)
			if err != nil {
				t.Fatalf("match %s for %s/%s: %v", name, test.goos, test.arch, err)
			}
			if matched {
				selected = append(selected, name)
			}
		}
		if len(selected) != 1 || selected[0] != test.want {
			t.Errorf("%s/%s selected %v, want [%s]", test.goos, test.arch, selected, test.want)
		}
	}
}

func TestLinuxReadDirAdaptersUseOneTargetSyscall(t *testing.T) {
	osDir := filepath.Join("..", "..", "std", "os")
	tests := []struct {
		name   string
		number string
	}{
		{"readdir_linux_amd64_renvo.go", "217"},
		{"readdir_linux_386_renvo.go", "220"},
		{"readdir_linux_aarch64_renvo.go", "61"},
		{"readdir_linux_arm_renvo.go", "217"},
	}
	for _, test := range tests {
		data, err := os.ReadFile(filepath.Join(osDir, test.name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "const renvoGetdents64 = "+test.number) {
			t.Errorf("%s does not select syscall %s", test.name, test.number)
		}
		if strings.Count(text, "syscall(") != 1 {
			t.Errorf("%s contains %d syscall calls, want one", test.name, strings.Count(text, "syscall("))
		}
	}
}

func TestGenericReadDirCodeHasNoSyscallTableProbing(t *testing.T) {
	osDir := filepath.Join("..", "..", "std", "os")
	paths := []string{
		filepath.Join(osDir, "os_renvo.go"),
		filepath.Join(osDir, "readdir_posix_renvo.go"),
		"renvo.go",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, forbidden := range []string{"getdents", "syscall(217", "syscall(220", "syscall(61", "pos+16", "pos+18", "pos+19"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("generic file %s contains target-specific detail %q", path, forbidden)
			}
		}
	}
}
