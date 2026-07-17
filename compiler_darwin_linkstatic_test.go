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

func TestDarwinArm64SelfHostedLinkStaticNamesSurvivePackageUnits(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skipf("Darwin self-host linkstatic test requires darwin/arm64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	files, err := getCompilerFiles(targetConfig{os: "darwin", arch: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	target := compilerTarget{name: "darwin/arm64", files: files}
	outDir := t.TempDir()
	backend := buildStage2Compiler(t, target, outDir)
	caseDir := filepath.Join(root, "rtg_tests", "regressions", "darwin_linkstatic_selfhost")
	frontend := filepath.Join(outDir, "frontend")
	buildFrontend := exec.Command("go", "build", "-o", frontend, "./rtg/cmd/rtg")
	buildFrontend.Dir = root
	if got, err := buildFrontend.CombinedOutput(); err != nil {
		t.Fatalf("frontend build failed: %v\n%s", err, got)
	}
	output := filepath.Join(outDir, "app")
	compile := exec.Command(frontend, "-t", target.name, "-s", "-o", output, "./cmd/app")
	compile.Dir = caseDir
	compile.Env = append(os.Environ(), "RTG_BACKEND="+backend, "RTG_STDROOT="+filepath.Join(root, "rtg", "std"))
	if got, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("self-hosted graphics compile failed: %v\n%s", err, got)
	}
	got, err := exec.Command(output).CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted linkstatic output failed: %v\n%s", err, got)
	}
	if string(got) != "PASS\n" {
		t.Fatalf("self-hosted linkstatic output = %q", got)
	}
}
