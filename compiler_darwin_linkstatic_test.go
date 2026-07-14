package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDarwinArm64LinkStaticObjCRuntime(t *testing.T) {
	src := []byte(`package main

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_getClass
func objcGetClass(name string) int { return 0 }

func appMain() int {
	if objcGetClass("NSObject") != 0 {
		print("PASS\n")
	}
	return 0
}
`)
	data, ok := RtgCompileSourceToBytes(src, "darwin/arm64")
	if !ok {
		t.Fatal("RtgCompileSourceToBytes failed")
	}
	for _, want := range []string{"/usr/lib/libobjc.A.dylib", "_objc_getClass"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("Darwin image missing %q", want)
		}
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return
	}
	out := filepath.Join(t.TempDir(), "objc-linkstatic")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("compiled Objective-C linkstatic test failed: %v\n%s", err, string(got))
	}
	if string(got) != "PASS\n" {
		t.Fatalf("compiled Objective-C output = %q", string(got))
	}
}
