package driver

import (
	"strings"
	"testing"
)

func TestParseOptionsCorpusShape(t *testing.T) {
	options := ParseOptions([]string{"-t", "windows/386", "-s", "-windows-gui", "-o", "/tmp/app", "./cmd/app"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if options.Target != "windows/386" || options.Output != "/tmp/app" || options.Package != "./cmd/app" || !options.Strip || !options.WindowsGUI {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseOptionsDefaultsTarget(t *testing.T) {
	options := ParseOptions([]string{"-o", "app", "./cmd/app"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if options.Target != DefaultTarget {
		t.Fatalf("target = %q, want %q", options.Target, DefaultTarget)
	}
	if options.Mode != ModeExecutable {
		t.Fatalf("mode = %q, want %q", options.Mode, ModeExecutable)
	}
	if options.Strip {
		t.Fatal("strip defaulted true")
	}
	if options.ModuleLicense != DefaultModuleLicense {
		t.Fatalf("module license = %q, want %q", options.ModuleLicense, DefaultModuleLicense)
	}
}

func TestParseOptionsAllowsOptionsAfterPackage(t *testing.T) {
	options := ParseOptions([]string{"./cmd/app", "-o", "app", "-s", "-t", "wasi/wasm32"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if options.Package != "./cmd/app" || options.Output != "app" || options.Target != "wasi/wasm32" || !options.Strip {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseOptionsAcceptsExplicitGoFiles(t *testing.T) {
	options := ParseOptions([]string{"main.go", "-o", "app", "helper.go"})
	if !options.Ok || options.Package != "main.go" || len(options.Files) != 2 || options.Files[0] != "main.go" || options.Files[1] != "helper.go" {
		t.Fatalf("file-list options = %#v", options)
	}
}

func TestParseOptionsAcceptsMixedGoAndCFiles(t *testing.T) {
	options := ParseOptions([]string{"main.go", "-o", "app", "helper.c"})
	if !options.Ok || len(options.Files) != 2 || options.Files[1] != "helper.c" {
		t.Fatalf("mixed file-list options = %#v", options)
	}
}

func TestCommandHelpRequested(t *testing.T) {
	for _, args := range [][]string{nil, {"renvo"}, {"renvo", "--help"}} {
		if !CommandHelpRequested(args) {
			t.Fatalf("CommandHelpRequested(%q) = false", args)
		}
	}
	if CommandHelpRequested([]string{"renvo", "-o", "app", "."}) {
		t.Fatal("compile command requested help")
	}
	for _, want := range []string{"Usage: renvo", "renvo cc -c", "-o <file>", "-system <file.rtg>", "-mode=<mode>", "kernel-module", "object", "source files...", "Explicit .go and .c files", "Exactly the named files", "windows/amd64", "windows/arm64", "darwin/arm64", "wasi/wasm32", "vm/vm32"} {
		if !strings.Contains(HelpText, want) {
			t.Fatalf("HelpText missing %q", want)
		}
	}
}

func TestNormalizeCCompilerCommand(t *testing.T) {
	args := NormalizeCCompilerCommand([]string{"renvo", "cc", "-c", "hello.c", "-o", "hello.o"})
	want := []string{"renvo", "-c", "hello.c", "-o", "hello.o"}
	if len(args) != len(want) {
		t.Fatalf("normalized args = %#v", args)
	}
	for i := 0; i < len(want); i++ {
		if args[i] != want[i] {
			t.Fatalf("normalized args = %#v", args)
		}
	}
	options := ParseOptions(args[1:])
	if !options.Ok || options.Mode != ModeObject || options.Target != "linux/amd64" || options.Package != "hello.c" || options.Output != "hello.o" {
		t.Fatalf("C compiler options = %#v", options)
	}
}

func TestParseOptionsBuildTags(t *testing.T) {
	options := ParseOptions([]string{"-tags", "renvo_bundle,debug", "-tags", "debug", "-o", "app", "./cmd/app"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if len(options.Tags) != 2 || options.Tags[0] != "renvo_bundle" || options.Tags[1] != "debug" {
		t.Fatalf("tags = %#v", options.Tags)
	}
}

func TestParseOptionsEmitUnit(t *testing.T) {
	options := ParseOptions([]string{"-emit-unit", "-o", "program.unit", "./cmd/app"})
	if !options.Ok || !options.EmitUnit || options.Output != "program.unit" {
		t.Fatalf("emit-unit options = %#v", options)
	}
}

func TestParseOptionsKernelModule(t *testing.T) {
	options := ParseOptions([]string{"-mode=kernel-module", "-o", "hello.ko", "hello.go"})
	if !options.Ok || options.Mode != ModeKernelModule || options.Target != "linux/amd64" {
		t.Fatalf("kernel module options = %#v", options)
	}
	if backendTargetForOptions(options.Target, options.Mode) != "linux-kernel/amd64" {
		t.Fatalf("backend target = %q", backendTargetForOptions(options.Target, options.Mode))
	}
}

func TestParseOptionsRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		err  int
		arg  string
		at   int
	}{
		{name: "missing output argument", args: []string{"-o"}, err: ParseErrMissingOutput, arg: "-o", at: 0},
		{name: "missing output option", args: []string{"./cmd/app"}, err: ParseErrMissingOutput, arg: "-o", at: 1},
		{name: "missing target argument", args: []string{"-t"}, err: ParseErrMissingTarget, arg: "-t", at: 0},
		{name: "unsupported target", args: []string{"-t", "darwin/amd64", "-o", "app", "./cmd/app"}, err: ParseErrUnsupportedTarget, arg: "darwin/amd64", at: 1},
		{name: "unknown option", args: []string{"-x", "-o", "app", "./cmd/app"}, err: ParseErrUnknownOption, arg: "-x", at: 0},
		{name: "missing tags", args: []string{"-tags"}, err: ParseErrMissingTags, arg: "-tags", at: 0},
		{name: "empty tags", args: []string{"-tags", "", "-o", "app", "./cmd/app"}, err: ParseErrInvalidTags, arg: "", at: 1},
		{name: "invalid tags", args: []string{"-tags", "renvo-bundle", "-o", "app", "./cmd/app"}, err: ParseErrInvalidTags, arg: "renvo-bundle", at: 1},
		{name: "empty tag item", args: []string{"-tags", "renvo_bundle,,debug", "-o", "app", "./cmd/app"}, err: ParseErrInvalidTags, arg: "renvo_bundle,,debug", at: 1},
		{name: "missing package", args: []string{"-o", "app"}, err: ParseErrMissingPackage, arg: "", at: 2},
		{name: "extra package", args: []string{"-o", "app", "./cmd/app", "./other"}, err: ParseErrExtraPackage, arg: "./other", at: 3},
		{name: "mixed file list", args: []string{"-o", "app", "main.go", "./other"}, err: ParseErrMixedFileList, arg: "./other", at: 3},
		{name: "GUI subsystem on non-Windows target", args: []string{"-windows-gui", "-o", "app", "./cmd/app"}, err: ParseErrWindowsGUIRequiresWindows, arg: "linux/amd64", at: 0},
		{name: "missing mode", args: []string{"-mode=", "-o", "app", "main.go"}, err: ParseErrMissingMode, arg: "-mode=", at: 0},
		{name: "unsupported mode", args: []string{"-mode=firmware", "-o", "app", "main.go"}, err: ParseErrUnsupportedMode, arg: "firmware", at: 0},
		{name: "kernel module on non-Linux target", args: []string{"-mode=kernel-module", "-t", "windows/amd64", "-o", "app.ko", "main.go"}, err: ParseErrModeRequiresLinuxAmd64, arg: "windows/amd64", at: 0},
		{name: "object on non-Linux target", args: []string{"-c", "-t", "windows/amd64", "-o", "app.o", "main.c"}, err: ParseErrObjectRequiresLinuxAmd64, arg: "windows/amd64", at: 0},
		{name: "object package mode", args: []string{"-c", "-o", "app.o", "./cmd/app"}, err: ParseErrObjectFileCount, arg: "./cmd/app", at: 4},
		{name: "object multiple files", args: []string{"-c", "-o", "app.o", "main.c", "other.c"}, err: ParseErrObjectFileCount, arg: "main.c", at: 5},
		{name: "object Go file", args: []string{"-c", "-o", "app.o", "main.go"}, err: ParseErrObjectRequiresC, arg: "main.go", at: 4},
		{name: "C script", args: []string{"-script", "-o", "app", "main.c"}, err: ParseErrScriptRequiresGo, arg: "main.c", at: 4},
	}
	for i := 0; i < len(tests); i++ {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			options := ParseOptions(tc.args)
			if options.Ok {
				t.Fatalf("ParseOptions accepted %#v", tc.args)
			}
			if options.Error != tc.err || options.ErrorArg != tc.arg || options.ErrorAt != tc.at {
				t.Fatalf("error = %d arg %q at %d, want %d %q %d", options.Error, options.ErrorArg, options.ErrorAt, tc.err, tc.arg, tc.at)
			}
		})
	}
}

func TestIsSupportedTarget(t *testing.T) {
	supported := []string{
		"linux/amd64",
		"linux/386",
		"linux/aarch64",
		"linux/arm",
		"windows/amd64",
		"darwin/arm64",
		"windows/386",
		"windows/arm64",
		"wasi/wasm32",
		"browser/wasm32",
		"vm/vm32",
	}
	for i := 0; i < len(supported); i++ {
		if !IsSupportedTarget(supported[i]) {
			t.Fatalf("target %q was not supported", supported[i])
		}
	}
	if IsSupportedTarget("linux/wasm32") {
		t.Fatal("invalid target was supported")
	}
}
