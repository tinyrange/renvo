package driver

import (
	"strings"
	"testing"
)

func TestParseOptionsCorpusShape(t *testing.T) {
	options := ParseOptions([]string{"-t", "linux/386", "-s", "-o", "/tmp/app", "./cmd/app"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if options.Target != "linux/386" || options.Output != "/tmp/app" || options.Package != "./cmd/app" || !options.Strip {
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
	if options.Strip {
		t.Fatal("strip defaulted true")
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

func TestCommandHelpRequested(t *testing.T) {
	for _, args := range [][]string{nil, {"rtg"}, {"rtg", "--help"}} {
		if !CommandHelpRequested(args) {
			t.Fatalf("CommandHelpRequested(%q) = false", args)
		}
	}
	if CommandHelpRequested([]string{"rtg", "-o", "app", "."}) {
		t.Fatal("compile command requested help")
	}
	for _, want := range []string{"Usage: rtg", "-o <file>", "windows/amd64", "darwin/arm64", "wasi/wasm32"} {
		if !strings.Contains(HelpText, want) {
			t.Fatalf("HelpText missing %q", want)
		}
	}
}

func TestParseOptionsBuildTags(t *testing.T) {
	options := ParseOptions([]string{"-tags", "rtg_bundle,debug", "-tags", "debug", "-o", "app", "./cmd/app"})
	if !options.Ok {
		t.Fatalf("ParseOptions failed: err=%d arg=%q at=%d", options.Error, options.ErrorArg, options.ErrorAt)
	}
	if len(options.Tags) != 2 || options.Tags[0] != "rtg_bundle" || options.Tags[1] != "debug" {
		t.Fatalf("tags = %#v", options.Tags)
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
		{name: "invalid tags", args: []string{"-tags", "rtg-bundle", "-o", "app", "./cmd/app"}, err: ParseErrInvalidTags, arg: "rtg-bundle", at: 1},
		{name: "empty tag item", args: []string{"-tags", "rtg_bundle,,debug", "-o", "app", "./cmd/app"}, err: ParseErrInvalidTags, arg: "rtg_bundle,,debug", at: 1},
		{name: "missing package", args: []string{"-o", "app"}, err: ParseErrMissingPackage, arg: "", at: 2},
		{name: "extra package", args: []string{"-o", "app", "./cmd/app", "./other"}, err: ParseErrExtraPackage, arg: "./other", at: 3},
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
		"wasi/wasm32",
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
