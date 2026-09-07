//go:build !renvo

package testfront

import (
	"reflect"
	"testing"
)

func TestRenvoGenerationSelectsTargetAndRequestedTags(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "impl_host.go", "//go:build !renvo\n\npackage tagged\nfunc value() int { return 1 }\n")
	writeTestFile(t, dir, "impl_renvo.go", "//go:build renvo\n\npackage tagged\nfunc value() int { return 2 }\n")
	writeTestFile(t, dir, "host_test.go", "//go:build !renvo\n\npackage tagged\nimport \"testing\"\nfunc TestHost(t *testing.T) {}\n")
	writeTestFile(t, dir, "native_test.go", "//go:build renvo\n\npackage tagged\nimport \"testing\"\nfunc TestNative(t *testing.T) {}\n")
	writeTestFile(t, dir, "extra_test.go", "//go:build renvo && extra\n\npackage tagged\nimport \"testing\"\nfunc TestExtra(t *testing.T) {}\n")
	native, err := GenerateRenvoPackageWithTags(dir, []string{"extra"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(native.Tests, []string{"TestExtra", "TestNative"}) {
		t.Fatal("incorrect native test selection", native.Tests)
	}
	if !generatedFileContains(native.Files, "impl_renvo.go", []byte("return 2")) || generatedFileContains(native.Files, "impl_host.go", []byte("return 1")) {
		t.Fatal("incorrect implementation selection")
	}
	host, err := GeneratePackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(host.Tests, []string{"TestHost"}) {
		t.Fatal("host selection changed", host.Tests)
	}
}
