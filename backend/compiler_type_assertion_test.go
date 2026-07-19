package main

import (
	"os"
	"path/filepath"
	"testing"
)

var failedTypeAssertionProgram = []byte(`package main

func appMain(args []string) int {
	var value interface{} = 7
	_ = value.(string)
	return 0
}
`)

func TestFailedTypeAssertionDiagnostic(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			image, ok := RenvoCompileSourceToBytes(failedTypeAssertionProgram, target.name)
			if !ok {
				t.Fatal("failed to compile type assertion program")
			}
			output := filepath.Join(t.TempDir(), "failed-type-assertion")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runTargetCommand(t, target, output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "" || result.stderr != "panic: interface conversion failed\n" {
				t.Fatalf("type assertion panic mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}
