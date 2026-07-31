//go:build !renvo

package backendjit

import (
	"bytes"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

func TestExcludedFamily(t *testing.T) {
	excluded := excludedFamily(rtg.TargetDescriptor{Family: rtg.BackendFamilyNativeV1, ISA: "amd64"})
	for _, name := range []string{
		"compiler_rtg_generated_impl.go", "compiler_rtg_inactive_impl.go",
	} {
		if !excluded[name] {
			t.Errorf("%s was not excluded", name)
		}
	}
	for _, name := range []string{
		"compiler_common_impl.go", "compiler_runtime_impl.go", "compiler_aarch64_impl.go",
		"compiler_amd64_impl.go", "compiler_amd64_target_impl.go",
		"compiler_wasm32_impl.go", "compiler_rtg_adapter_impl.go", "compiler_unit_impl.go",
	} {
		if excluded[name] {
			t.Fatalf("%s was excluded from the prepared compiler kernel", name)
		}
	}
}

func TestPreparationBundleCarriesTypedSettings(t *testing.T) {
	foundMain := false
	foundPreparedMode := false
	for i := 0; i < backendcompiled.CompilerSourceCount; i++ {
		name, source, ok := backendcompiled.CompilerSource(i)
		if !ok {
			t.Fatalf("read embedded compiler source %d", i)
		}
		if name == "compiler_main.go" {
			foundMain = bytes.Contains([]byte(source), []byte("var renvoFixedTarget int = 0"))
		}
		if name == "compiler_target_policy_impl.go" {
			foundPreparedMode =
				bytes.Contains([]byte(source), []byte("const renvoPreparedBackend = 1"))
		}
	}
	if !foundMain || !foundPreparedMode {
		t.Fatalf("typed preparation inputs: main=%v mode=%v", foundMain, foundPreparedMode)
	}

	source, err := os.ReadFile("../../backend/compiler_target_policy_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source, []byte("const renvoPreparedBackend = 0")) {
		t.Fatal("checked-in compiler policy did not retain built-in mode")
	}
}

func TestPublishIsReusable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "entry.rtgb")
	if err := publish(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := publish(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != "second" {
		t.Fatalf("cache source = %q", source)
	}
}

func TestConcurrentPublishNeverExposesPartialEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "entry.rtgb")
	values := [][]byte{
		bytes.Repeat([]byte("a"), 4096),
		bytes.Repeat([]byte("b"), 8192),
		bytes.Repeat([]byte("c"), 16384),
	}
	var wait sync.WaitGroup
	for i := 0; i < len(values); i++ {
		wait.Add(1)
		go func(value []byte) {
			defer wait.Done()
			if err := publish(path, value); err != nil {
				t.Errorf("publish: %v", err)
			}
		}(values[i])
	}
	wait.Wait()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if bytes.Equal(got, value) {
			return
		}
	}
	t.Fatalf("published entry has partial size %d", len(got))
}

func TestCacheKeyPreservesExactTargetAndHostIdentity(t *testing.T) {
	var descriptor rtg.TargetDescriptor
	descriptor.Name = "a/b"
	left := cacheKey(descriptor, "linux/amd64")
	descriptor.Name = "a_b"
	right := cacheKey(descriptor, "linux/amd64")
	if left == right {
		t.Fatalf("target names share cache key %q", left)
	}
	descriptor.Name = "a/b"
	if left == cacheKey(descriptor, "linux_amd64") {
		t.Fatalf("host names share cache key %q", left)
	}
}

type memoryRunner struct {
	request Request
}

func (runner *memoryRunner) Run(_ rtgb.Artifact, request Request) driver.BackendResult {
	runner.request = request
	return driver.BackendResult{Binary: []byte("in-memory-output"), Ok: true}
}

func TestInMemoryRunnerUsesVersionedProtocol(t *testing.T) {
	var definition [32]byte
	definition[0] = 42
	descriptor := rtg.TargetDescriptor{
		Name: "example/amd64", Family: rtg.BackendFamilyNativeV1,
		Definition: definition, Version: rtg.DescriptorVersion,
	}
	base := make([]byte, 14)
	copy(base, unit.Magic)
	base[4] = byte(unit.Version)
	base[8] = byte(unit.TagUnit)
	bound, ok := unit.BindTarget(base, unit.TargetBinding{
		Target: descriptor.Name, Definition: string(definition[:]),
		DescriptorVersion: descriptor.Version,
	})
	if !ok {
		t.Fatal("BindTarget failed")
	}
	runner := new(memoryRunner)
	backend := &Backend{
		runner: runner,
		prepared: Prepared{Ok: true, Artifact: rtgb.Artifact{
			Descriptor: descriptor, Protocol: ProtocolVersion,
			Unit: unit.Version, Optimization: OptimizationVersion,
		}},
	}
	result := backend.CompileUnit(bound, descriptor.Name, true, false)
	if !result.Ok || string(result.Binary) != "in-memory-output" {
		t.Fatalf("in-memory compile = %#v", result)
	}
	if runner.request.Protocol != ProtocolVersion ||
		!bytes.Equal(runner.request.Unit, bound) || !runner.request.Options.Strip {
		t.Fatalf("runner request = %#v", runner.request)
	}
	backend.prepared.Artifact.Descriptor.Definition[0]++
	mismatch := backend.CompileUnit(bound, descriptor.Name, false, false)
	if mismatch.Ok || mismatch.Diagnostic.Code != "RENVO-BACKEND-007" {
		t.Fatalf("definition-mismatched unit = %#v", mismatch)
	}
}

func TestCompiledInBootstrapPreparesAndCachesBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	// Give the unknown architecture a semantically equivalent but observably
	// different zero-immediate encoding. The output assertion below proves the
	// prepared compiler executed this definition-owned encoder rather than an
	// advertised amd64 fallback.
	definition := copyNativeDefinition(t, root,
		"target linux/amd64 {", "target example/example64 {",
		"out.Bytes2(0x31, 0xc0)",
		"out.Bytes2(0x6a, 0)\n\t\t\temitOpcodeRegister(out, 0x58, destination)")
	stdRoot := filepath.Join(root, "std")
	source := filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go")
	cache := t.TempDir()
	args := []string{
		"-backend", definition,
		"-t", "example/example64",
		"-s",
		"-o", "app",
		source,
	}

	first := New(definition, filepath.Join(root, "backend"), stdRoot, cache, backendcompiled.Backend{})
	firstResult := driver.CompileFromFS(args, root, stdRoot, driver.OSFS{}, first)
	if !firstResult.Ok {
		t.Fatalf("cold custom backend compile failed: %#v", firstResult.Diagnostic)
	}
	if first.prepared.CacheHit {
		t.Fatal("cold custom backend unexpectedly hit cache")
	}
	if !bytes.HasPrefix(firstResult.Binary, []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("cold custom backend output prefix = % x", firstResult.Binary[:minInt(4, len(firstResult.Binary))])
	}
	if !bytes.Contains(firstResult.Binary, []byte{0x6a, 0, 0x58}) {
		t.Fatal("custom backend output does not contain the definition-owned zero-immediate encoding")
	}
	executable := filepath.Join(t.TempDir(), "custom-backend-output")
	if err := os.WriteFile(executable, firstResult.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	execution, err := exec.Command(executable).CombinedOutput()
	if err != nil {
		t.Fatalf("custom backend output failed: %v\n%s", err, execution)
	}
	if string(execution) != "PASS\n" {
		t.Fatalf("custom backend output = %q, want PASS", execution)
	}
	entryArgs := append([]string(nil), args...)
	entryArgs[len(entryArgs)-1] = filepath.Join(root, "internal", "backendjit", "testdata", "args_env.go")
	entryResult := driver.CompileFromFS(entryArgs, root, stdRoot, driver.OSFS{}, first)
	if !entryResult.Ok {
		t.Fatalf("custom backend entry compile failed: %#v", entryResult.Diagnostic)
	}
	entryExecutable := filepath.Join(t.TempDir(), "custom-backend-entry-output")
	if err := os.WriteFile(entryExecutable, entryResult.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	entryCommand := exec.Command(entryExecutable, "definition-owned-entry")
	entryCommand.Env = []string{"PATH=/definition-owned-entry"}
	entryExecution, err := entryCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("custom backend entry output failed: %v\n%s", err, entryExecution)
	}
	if string(entryExecution) != "PASS\n" {
		t.Fatalf("custom backend entry output = %q, want PASS", entryExecution)
	}
	equalityArgs := append([]string(nil), args...)
	equalityArgs[len(equalityArgs)-1] = filepath.Join(root, "backend", "tests", "prepared_backend_equality_branch.go")
	equalityResult := driver.CompileFromFS(equalityArgs, root, stdRoot, driver.OSFS{}, first)
	if !equalityResult.Ok {
		t.Fatalf("custom backend equality compile failed: %#v", equalityResult.Diagnostic)
	}
	equalityExecutable := filepath.Join(t.TempDir(), "custom-backend-equality-output")
	if err := os.WriteFile(equalityExecutable, equalityResult.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	equalityExecution, err := exec.Command(equalityExecutable, "one-argument").CombinedOutput()
	if err != nil {
		t.Fatalf("custom backend equality output failed: %v\n%s", err, equalityExecution)
	}
	if string(equalityExecution) != "PASS\n" {
		t.Fatalf("custom backend equality output = %q, want PASS", equalityExecution)
	}
	for _, mutate := range []func(*rtgb.Artifact){
		func(artifact *rtgb.Artifact) { artifact.Protocol++ },
		func(artifact *rtgb.Artifact) { artifact.Unit++ },
		func(artifact *rtgb.Artifact) { artifact.Descriptor.Version++ },
		func(artifact *rtgb.Artifact) { artifact.Optimization++ },
		func(artifact *rtgb.Artifact) { artifact.Generator++ },
		func(artifact *rtgb.Artifact) { artifact.Kernel++ },
		func(artifact *rtgb.Artifact) { artifact.Host += "-wrong" },
	} {
		artifact := first.prepared.Artifact
		mutate(&artifact)
		encoded, ok := rtgb.Encode(artifact)
		if !ok {
			t.Fatal("could not encode incompatible artifact")
		}
		if loaded := Load(encoded); loaded.Ok {
			t.Fatalf("accepted incompatible artifact %#v", artifact)
		}
	}

	second := New(definition, filepath.Join(root, "backend"), stdRoot, cache, backendcompiled.Backend{})
	secondResult := driver.CompileFromFS(args, root, stdRoot, driver.OSFS{}, second)
	if !secondResult.Ok {
		t.Fatalf("warm custom backend compile failed: %#v", secondResult.Diagnostic)
	}
	if !second.prepared.CacheHit {
		t.Fatal("warm custom backend did not hit cache")
	}
	if !bytes.Equal(firstResult.Binary, secondResult.Binary) {
		t.Fatal("cold and warm custom backend outputs differ")
	}
}

func TestCompiledInBootstrapUsesAArch64Definition(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := copyNativeDefinition(t, root,
		"target linux/aarch64 {", "target example/aarch64 {")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "example/aarch64",
		"-s",
		"-o", "app",
		filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("custom AArch64 backend compile failed: %#v", result.Diagnostic)
	}
	if !bytes.HasPrefix(result.Binary, []byte{0x7f, 'E', 'L', 'F'}) ||
		len(result.Binary) < 20 || result.Binary[18] != 183 {
		t.Fatalf("custom AArch64 output header = % x", result.Binary[:minInt(20, len(result.Binary))])
	}
}

func TestCompiledInBootstrapUsesDefinitionOwnedPEImages(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		file    string
		target  string
		custom  string
		machine []byte
	}{
		{"native.rtg", "windows/amd64", "example/windows-amd64", []byte{0x64, 0x86}},
		{"native.rtg", "windows/386", "example/windows-386", []byte{0x4c, 0x01}},
		{"native.rtg", "windows/arm64", "example/windows-arm64", []byte{0x64, 0xaa}},
	}
	for _, test := range cases {
		t.Run(test.custom, func(t *testing.T) {
			definition := copyNativeDefinition(t, root,
				"target "+test.target+" {", "target "+test.custom+" {")
			result := driver.CompileFromFS([]string{
				"-backend", definition,
				"-t", test.custom,
				"-s",
				"-o", "app.exe",
				filepath.Join(root, "internal", "backendjit", "testdata", "windows_runtime.go"),
			}, root, filepath.Join(root, "std"), driver.OSFS{},
				New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
					t.TempDir(), backendcompiled.Backend{}))
			if !result.Ok {
				t.Fatalf("custom Windows backend compile failed: %#v", result.Diagnostic)
			}
			if len(result.Binary) < 0x100 ||
				!bytes.HasPrefix(result.Binary, []byte{'M', 'Z'}) ||
				result.Binary[0x80] != 'P' || result.Binary[0x81] != 'E' ||
				!bytes.Equal(result.Binary[0x84:0x86], test.machine) {
				t.Fatalf("custom PE output header = % x", result.Binary[:minInt(0x90, len(result.Binary))])
			}
			if !bytes.Contains(result.Binary, []byte("DefinitionStaticImport\x00")) {
				t.Fatal("custom PE output is missing its definition-owned static import")
			}
		})
	}
}

func TestCompiledInBootstrapUsesDefinitionOwnedKernelModuleImage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s",
			runtime.GOOS, runtime.GOARCH)
	}
	btf, err := os.ReadFile("/sys/kernel/btf/vmlinux")
	if err != nil || len(btf) == 0 {
		t.Skip("host kernel BTF is unavailable")
	}
	release, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		t.Skip("host kernel release is unavailable")
	}
	release = bytes.TrimSpace(release)
	symvers, err := os.ReadFile(filepath.Join(
		"/lib/modules", string(release), "build", "Module.symvers"))
	if err != nil || len(symvers) == 0 {
		t.Skip("host kernel Module.symvers is unavailable")
	}
	if !bytes.Contains(symvers, []byte("\t_printk\t")) {
		t.Skip("host kernel does not expose _printk")
	}
	if !bytes.Contains(symvers, []byte("\tfor_each_kernel_tracepoint\t")) {
		t.Skip("host kernel does not expose for_each_kernel_tracepoint")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := copyNativeDefinition(t, root,
		"target linux-kernel/amd64 {", "target example/kernel-amd64 {")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "example/kernel-amd64",
		"-mode=kernel-module",
		"-o", "prepared.ko",
		filepath.Join(root, "internal", "backendjit", "testdata", "kernel_runtime.go"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("custom kernel backend compile failed: %#v", result.Diagnostic)
	}
	image, err := elf.NewFile(bytes.NewReader(result.Binary))
	if err != nil {
		t.Fatalf("parse custom kernel module: %v", err)
	}
	if image.Type != elf.ET_REL || image.Machine != elf.EM_X86_64 {
		t.Fatalf("custom kernel image type/machine = %v/%v",
			image.Type, image.Machine)
	}
	for _, name := range []string{
		".text", ".rela.text", ".modinfo", ".gnu.linkonce.this_module",
		".rela.gnu.linkonce.this_module",
	} {
		if image.Section(name) == nil {
			t.Fatalf("custom kernel image is missing %s", name)
		}
	}
	symbols, err := image.Symbols()
	if err != nil {
		t.Fatalf("read custom kernel symbols: %v", err)
	}
	required := map[string]bool{
		"init_module": false, "cleanup_module": false,
		"__this_module": false, "_printk": false,
		"for_each_kernel_tracepoint": false,
	}
	for i := 0; i < len(symbols); i++ {
		if _, found := required[symbols[i].Name]; found {
			required[symbols[i].Name] = true
		}
	}
	for name, found := range required {
		if !found {
			t.Fatalf("custom kernel image is missing symbol %s", name)
		}
	}
}

func TestCompiledInBootstrapUsesDefinitionOwnedMachOImage(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := copyNativeDefinition(t, root,
		"target darwin/arm64 {", "target example/darwin-arm64 {")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "example/darwin-arm64",
		"-s",
		"-o", "app",
		filepath.Join(root, "internal", "backendjit", "testdata", "minimal.go"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("custom Darwin backend compile failed: %#v", result.Diagnostic)
	}
	if len(result.Binary) < 0x1000 ||
		!bytes.HasPrefix(result.Binary, []byte{0xcf, 0xfa, 0xed, 0xfe}) ||
		!bytes.Equal(result.Binary[4:8], []byte{0x0c, 0, 0, 1}) {
		t.Fatalf("custom Mach-O output header = % x",
			result.Binary[:minInt(32, len(result.Binary))])
	}
	if !bytes.Contains(result.Binary, []byte("_exit\x00")) ||
		!bytes.Contains(result.Binary, []byte("/usr/lib/libSystem.B.dylib\x00")) {
		t.Fatal("custom Mach-O output is missing definition-owned imports")
	}
	if !bytes.Contains(result.Binary, []byte{0xfa, 0xde, 0x0c, 0xc0}) {
		t.Fatal("custom Mach-O output is missing its ad-hoc code signature")
	}
}

func TestCompiledInBootstrapUses32BitDefinitions(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		file    string
		target  string
		custom  string
		machine byte
	}{
		{"native.rtg", "linux/386", "example/386", 3},
		{"native.rtg", "linux/arm", "example/arm", 40},
	}
	for _, test := range cases {
		t.Run(test.custom, func(t *testing.T) {
			definition := copyNativeDefinition(t, root,
				"target "+test.target+" {", "target "+test.custom+" {")
			result := driver.CompileFromFS([]string{
				"-backend", definition,
				"-t", test.custom,
				"-s",
				"-o", "app",
				filepath.Join(root, "internal", "backendjit", "testdata", "semantic_runtime.go"),
			}, root, filepath.Join(root, "std"), driver.OSFS{},
				New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
					t.TempDir(), backendcompiled.Backend{}))
			if !result.Ok {
				t.Fatalf("custom backend compile failed: %#v", result.Diagnostic)
			}
			if !bytes.HasPrefix(result.Binary, []byte{0x7f, 'E', 'L', 'F'}) ||
				len(result.Binary) < 20 || result.Binary[18] != test.machine {
				t.Fatalf("custom output header = % x", result.Binary[:minInt(20, len(result.Binary))])
			}
		})
	}
}

func TestCompiledInBootstrapUsesVM32Definitions(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		target string
		custom string
		magic  []byte
	}{
		{"wasi/wasm32", "example/wasm32", []byte{0, 'a', 's', 'm'}},
		{"vm/vm32", "example/vm32", []byte{'R', 'N', 'V', 'B'}},
	}
	for _, test := range cases {
		t.Run(test.custom, func(t *testing.T) {
			definitionSource, readErr := os.ReadFile(filepath.Join(
				root, "backend", "definitions", "wasm32.rtg"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			definitionSource = bytes.Replace(definitionSource,
				[]byte("target "+test.target+" {"),
				[]byte("target "+test.custom+" {"), 1)
			definition := filepath.Join(t.TempDir(), "example-vm32.rtg")
			if writeErr := os.WriteFile(definition, definitionSource, 0o600); writeErr != nil {
				t.Fatal(writeErr)
			}
			result := driver.CompileFromFS([]string{
				"-backend", definition,
				"-t", test.custom,
				"-arena-size", "8192",
				"-s",
				"-o", "app",
				filepath.Join(root, "internal", "backendjit", "testdata", "semantic_runtime.go"),
			}, root, filepath.Join(root, "std"), driver.OSFS{},
				New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
					t.TempDir(), backendcompiled.Backend{}))
			if !result.Ok {
				t.Fatalf("custom VM32 backend compile failed: %#v", result.Diagnostic)
			}
			if !bytes.HasPrefix(result.Binary, test.magic) {
				t.Fatalf("custom output prefix = % x", result.Binary[:minInt(8, len(result.Binary))])
			}
			if test.custom == "example/vm32" {
				execution := vm.Run(result.Binary, vm.Limits{
					Steps:  100000,
					Memory: 32768,
				})
				if execution.Trap != vm.TrapNone || execution.ExitCode != 0 ||
					string(execution.Output) != "PASS\n" {
					t.Fatalf("custom VM32 execution = exit %d, trap %d at pc %d opcode %d, output %q, steps %d",
						execution.ExitCode, execution.Trap, execution.TrapPC, execution.TrapOpcode,
						execution.Output, execution.Steps)
				}
			}
		})
	}
}

func copyNativeDefinition(t *testing.T, root string, replacements ...string) string {
	t.Helper()
	if len(replacements)%2 != 0 {
		t.Fatal("copyNativeDefinition requires old/new replacement pairs")
	}
	type definitionFile struct {
		name   string
		source []byte
	}
	names := []string{
		"native.rtg",
		"native/x86_64.rtg",
		"native/x86_32.rtg",
		"native/aarch64.rtg",
		"native/arm.rtg",
	}
	files := make([]definitionFile, len(names))
	sourceRoot := filepath.Join(root, "backend", "definitions")
	for i := 0; i < len(names); i++ {
		source, err := os.ReadFile(filepath.Join(sourceRoot, names[i]))
		if err != nil {
			t.Fatal(err)
		}
		files[i] = definitionFile{name: names[i], source: source}
	}
	for replacement := 0; replacement < len(replacements); replacement += 2 {
		old := []byte(replacements[replacement])
		newText := []byte(replacements[replacement+1])
		found := false
		for i := 0; i < len(files); i++ {
			if !bytes.Contains(files[i].source, old) {
				continue
			}
			if found || bytes.Count(files[i].source, old) != 1 {
				t.Fatalf("native definition replacement %q is not unique", old)
			}
			files[i].source = bytes.Replace(files[i].source, old, newText, 1)
			found = true
		}
		if !found {
			t.Fatalf("native definition replacement %q was not found", old)
		}
	}
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(destination, "native"), 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(files); i++ {
		if err := os.WriteFile(filepath.Join(destination, files[i].name),
			files[i].source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(destination, "native.rtg")
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
