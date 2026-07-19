package main

import (
	"path/filepath"
	"testing"
)

func TestRuntimeIntrinsicFingerprintTable(t *testing.T) {
	tests := []struct {
		name string
		id   int
	}{
		{name: "renvo_runtime_Exit", id: 1},
		{name: "renvo_runtime_ArenaMark", id: 2},
		{name: "renvo_runtime_ArenaReset", id: 3},
		{name: "renvo_runtime_ArenaPersistMark", id: 4},
		{name: "renvo_runtime_ArenaPersistReset", id: 5},
		{name: "renvo_runtime_ArenaPersistString", id: 6},
		{name: "renvo_runtime_ArenaPersistBytes", id: 7},
		{name: "renvo_runtime_ArenaPersistCheckBools", id: 8},
		{name: "renvo_runtime_ArenaPersistCheckNameRefs", id: 8},
		{name: "renvo_runtime_ArenaPersistCheckSelectorRefs", id: 8},
		{name: "renvo_runtime_ArenaPersistCheckTypeRefs", id: 8},
		{name: "renvo_runtime_ArenaDiscard", id: 12},
		{name: "renvo_runtime_ArenaDiscardBytes", id: 13},
		{name: "renvo_runtime_ArenaDiscardLinkTokens", id: 13},
		{name: "renvo_runtime_ArenaDiscardLowerTokens", id: 13},
		{name: "renvo_runtime_ArenaDiscardUnitTokens", id: 13},
	}

	for _, test := range tests {
		source := []byte("prefix" + test.name + "suffix")
		if got := renvoRuntimeIntrinsicID(source, len("prefix"), len(source)-len("suffix")); got != test.id {
			t.Errorf("%s: got intrinsic %d, want %d", test.name, got, test.id)
		}
		nearMiss := []byte(test.name + "x")
		if got := renvoRuntimeIntrinsicID(nearMiss, 0, len(nearMiss)); got != 0 {
			t.Errorf("%s near miss: got intrinsic %d, want 0", test.name, got)
		}
	}
}

func TestArenaAllocationBoundsAndRecovery(t *testing.T) {
	targets := supportedCompilerTargets(t)
	if len(targets) == 0 {
		t.Fatal("no native compiler target")
	}
	target := targets[0]
	skipIfTargetRunnerMissing(t, target)
	outDir := t.TempDir()
	stage2 := buildStage2Compiler(t, target, outDir)

	tests := []struct {
		name      string
		source    string
		arenaSize string
		wantOut   string
		wantErr   string
		wantExit  int
	}{
		{name: "below limit", source: "tests/arena_allocation_below.renvo", arenaSize: "4096", wantOut: "PASS\n"},
		{name: "at limit", source: "tests/arena_allocation_boundary.renvo", arenaSize: "4096", wantOut: "PASS\n"},
		{name: "make recover", source: "tests/arena_oom_make_recover.renvo", arenaSize: "4096", wantOut: "PASS\n"},
		{name: "append recover", source: "tests/arena_oom_append_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "string recover", source: "tests/arena_oom_string_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "new recover", source: "tests/arena_oom_new_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "interface recover", source: "tests/arena_oom_interface_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "closure recover", source: "tests/arena_oom_closure_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "defer recover", source: "tests/arena_oom_defer_recover.renvo", arenaSize: "256", wantOut: "PASS\n"},
		{name: "uncaught", source: "tests/arena_oom_uncaught.renvo", arenaSize: "4096", wantErr: "panic: out of memory\n", wantExit: 2},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			outputFile := filepath.Join(outDir, "arena-"+test.name)
			compiled, err := runTargetCommand(t, target, stage2,
				"-t", target.name,
				"-arena-size", test.arenaSize,
				"-o", outputFile,
				test.source)
			if err != nil {
				t.Fatalf("compiler execution failed: %v", err)
			}
			if compiled.exitCode != 0 {
				t.Fatalf("compilation failed with exit code %d\nstdout: %sstderr: %s", compiled.exitCode, compiled.stdout, compiled.stderr)
			}
			actual, err := runTargetCommand(t, target, outputFile)
			if err != nil {
				t.Fatalf("output execution failed: %v", err)
			}
			if actual.stdout != test.wantOut || actual.stderr != test.wantErr || actual.exitCode != test.wantExit {
				t.Fatalf("output mismatch\nstdout: got %q, want %q\nstderr: got %q, want %q\nexit: got %d, want %d",
					actual.stdout, test.wantOut, actual.stderr, test.wantErr, actual.exitCode, test.wantExit)
			}
		})
	}
}
