//go:build !renvo

package driver

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseOptionsArenaPolicy(t *testing.T) {
	options := ParseOptions([]string{"-arena-size", "65536", "-o", "app", "./cmd/app"})
	if !options.Ok || options.ArenaSize != 65536 {
		t.Fatalf("arena options = %#v", options)
	}
	for _, test := range []struct {
		args []string
		err  int
	}{
		{args: []string{"-arena-size"}, err: ParseErrMissingArenaSize},
		{args: []string{"-arena-size", "255", "-o", "app", "./cmd/app"}, err: ParseErrInvalidArenaSize},
		{args: []string{"-arena-size", "1073741825", "-o", "app", "./cmd/app"}, err: ParseErrInvalidArenaSize},
	} {
		got := ParseOptions(test.args)
		if got.Ok || got.Error != test.err {
			t.Errorf("ParseOptions(%q) = %#v, want error %d", test.args, got, test.err)
		}
	}
	if !strings.Contains(HelpText, "-arena-size") {
		t.Fatal("command help does not document arena policy")
	}
}

type recordingArenaBackend struct {
	arenaSize int
}

func (b *recordingArenaBackend) CompileUnit([]byte, string, bool, bool) BackendResult {
	return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "TEST-ARENA-001", Message: "arena-aware entrypoint was not used"}}
}

func (b *recordingArenaBackend) CompileUnitWithArena(_ []byte, _ string, _ bool, _ bool, arenaSize int) BackendResult {
	b.arenaSize = arenaSize
	return BackendResult{Binary: []byte("binary"), Ok: true}
}

func TestCompilePassesArenaPolicyToEmbeddedBackend(t *testing.T) {
	backend := &recordingArenaBackend{}
	result := CompileUnit([]string{"-arena-size", "131072", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), backend)
	if !result.Ok {
		t.Fatalf("CompileUnit failed: %#v", result)
	}
	if backend.arenaSize != 131072 {
		t.Fatalf("backend arena = %d, want 131072", backend.arenaSize)
	}
}

func TestCompileRejectsArenaPolicyForLegacyBackend(t *testing.T) {
	backend := &recordingBackend{binary: []byte("binary")}
	result := CompileUnit([]string{"-arena-size", "131072", "-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles(), backend)
	if result.Ok || result.Diagnostic.Code != "RENVO-BACKEND-005" {
		t.Fatalf("legacy backend result = %#v", result)
	}
}

func TestCommandBackendForwardsArenaPolicy(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	built := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles())
	if !built.Ok {
		t.Fatalf("BuildUnit failed: %#v", built)
	}
	backend := CommandBackend{
		Path: executable,
		Args: []string{"-test.run=TestArenaCommandBackendHelper", "--"},
		Env:  []string{"RENVO_DRIVER_ARENA_HELPER=1"},
	}
	result := backend.CompileUnitWithArena(built.Unit, "linux/amd64", false, false, 262144)
	if !result.Ok || string(result.Binary) != "PASS\n" {
		t.Fatalf("command backend result = %#v", result)
	}
}

func TestArenaCommandBackendHelper(t *testing.T) {
	if os.Getenv("RENVO_DRIVER_ARENA_HELPER") != "1" {
		return
	}
	args := helperBackendArgs(os.Args)
	found := false
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-arena-size" && args[i+1] == "262144" {
			found = true
			break
		}
	}
	if !found {
		os.Exit(2)
	}
	if _, err := io.ReadAll(os.Stdin); err != nil {
		os.Exit(3)
	}
	_, _ = os.Stdout.Write([]byte("PASS\n"))
	os.Exit(0)
}
