//go:build !renvo

package backendjit

import (
	"bytes"
	"debug/elf"
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"sync"
	"testing"

	"renvo.dev/internal/backendbuiltin"
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
				bytes.Contains([]byte(source), []byte("const renvoPreparedBackendActive = 1"))
		}
	}
	if !foundMain || !foundPreparedMode {
		t.Fatalf("typed preparation inputs: main=%v mode=%v", foundMain, foundPreparedMode)
	}

	source, err := os.ReadFile("../../backend/compiler_target_policy_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source, []byte("const renvoPreparedBackendActive = 0")) {
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

func TestSeededCacheKeyCoversHostTargetAndCompatibilityIdentity(t *testing.T) {
	descriptor := rtg.TargetDescriptor{Name: "target/example", Version: rtg.DescriptorVersion}
	descriptor.Definition[0] = 1
	var hostDefinition [32]byte
	hostDefinition[0] = 2
	base := seededCacheKey("compiler", seededKernelVersion, descriptor, "host/example", hostDefinition)

	targetChanged := descriptor
	targetChanged.Definition[0]++
	if base == seededCacheKey("compiler", seededKernelVersion, targetChanged, "host/example", hostDefinition) {
		t.Fatal("target semantic digest did not change the compiler cache key")
	}
	hostChanged := hostDefinition
	hostChanged[0]++
	if base == seededCacheKey("compiler", seededKernelVersion, descriptor, "host/example", hostChanged) {
		t.Fatal("host semantic digest did not change the compiler cache key")
	}
	if base == seededCacheKey("compiler", seededKernelVersion+1, descriptor, "host/example", hostDefinition) {
		t.Fatal("kernel compatibility version did not change the compiler cache key")
	}
	if base == seededCacheKey("compiler", seededKernelVersion, descriptor, "host_example", hostDefinition) {
		t.Fatal("host canonical name did not change the compiler cache key")
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

func TestCompilerRequestArgsPreserveOptions(t *testing.T) {
	got := compilerRequestArgs("example/amd64", driver.BackendCompileOptions{
		Strip: true, WindowsGUI: true, EmitImage: true, ObjectFile: true,
		ArenaSize: 4096, Output: "module-name", ModuleLicense: "Dual MIT/GPL",
	}, "-", "-")
	want := []string{
		"-t", "example/amd64", "-s", "-windows-gui", "-emit-image", "-object",
		"-arena-size", "4096", "-module-name", "module-name",
		"-module-license", "Dual MIT/GPL", "-o", "-", "-",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("compiler process arguments = %#v, want %#v", got, want)
	}
}

func TestVM32SeedPromotesExternalCompilerAndCachesExecutable(t *testing.T) {
	if os.Getenv("RENVO_SEEDED_INTEGRATION") != "1" {
		t.Skip("set RENVO_SEEDED_INTEGRATION=1 to run the cold VM32 seed promotion integration")
	}
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
		"if destination.Code == 0 && value == 0 {\n\t\t\tout.Bytes2(0x31, 0xc0)",
		"if destination.Code == 0 && value == 0 {\n\t\t\tout.Bytes2(0x6a, 0)\n\t\t\temitOpcodeRegister(out, 0x58, destination)")
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

	first := NewSeeded(definition, stdRoot, cache)
	firstResult := driver.CompileFromFS(args, root, stdRoot, driver.OSFS{}, first)
	if !firstResult.Ok {
		t.Fatalf("cold custom backend compile failed: %#v", firstResult.Diagnostic)
	}
	if first.prepared.CacheHit {
		t.Fatal("cold custom backend unexpectedly hit cache")
	}
	if first.prepared.ExecutablePath == "" || !trustedCachedExecutable(first.prepared.ExecutablePath) {
		t.Fatalf("cold custom backend did not publish a trusted executable: host=%q path=%q diagnostic=%#v", first.prepared.Artifact.Host, first.prepared.ExecutablePath, first.prepared.Diagnostic)
	}
	binding, ok := unit.ReadTargetBinding(firstResult.Build.Unit)
	if !ok {
		t.Fatal("cold custom backend unit has no target binding")
	}
	wrongDefinition := []byte(binding.Definition)
	wrongDefinition[0] ^= 0xff
	for _, mismatch := range []struct {
		name    string
		binding unit.TargetBinding
	}{
		{"target", unit.TargetBinding{Target: binding.Target + "-wrong", Definition: binding.Definition, DescriptorVersion: binding.DescriptorVersion}},
		{"definition", unit.TargetBinding{Target: binding.Target, Definition: string(wrongDefinition), DescriptorVersion: binding.DescriptorVersion}},
		{"descriptor version", unit.TargetBinding{Target: binding.Target, Definition: binding.Definition, DescriptorVersion: binding.DescriptorVersion + 1}},
	} {
		mismatchedUnit, bound := unit.BindTarget(firstResult.Build.Unit, mismatch.binding)
		if !bound {
			t.Fatalf("could not construct %s-mismatched unit", mismatch.name)
		}
		mismatched := runNativeCompiler(first.prepared.ExecutablePath, first.prepared.Artifact, Request{
			Protocol: ProtocolVersion, Unit: mismatchedUnit,
			Options: driver.BackendCompileOptions{Target: "example/example64", Strip: true},
		})
		if mismatched.Ok || len(mismatched.Binary) != 0 || mismatched.Diagnostic.Code != "RENVO-BACKEND-009" {
			t.Fatalf("%s-mismatched compiler request: ok=%v bytes=%d diagnostic=%#v", mismatch.name, mismatched.Ok, len(mismatched.Binary), mismatched.Diagnostic)
		}
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
	immediateArgs := append([]string(nil), args...)
	immediateArgs[len(immediateArgs)-1] = filepath.Join(root, "backend", "tests", "prepared_push_immediate_argument.go")
	immediateResult := driver.CompileFromFS(immediateArgs, root, stdRoot, driver.OSFS{}, first)
	if !immediateResult.Ok {
		t.Fatalf("custom backend immediate argument compile failed: %#v", immediateResult.Diagnostic)
	}
	immediateExecutable := filepath.Join(t.TempDir(), "custom-backend-immediate-argument-output")
	if err := os.WriteFile(immediateExecutable, immediateResult.Binary, 0o700); err != nil {
		t.Fatal(err)
	}
	immediateExecution, err := exec.Command(immediateExecutable).CombinedOutput()
	if err != nil {
		t.Fatalf("custom backend immediate argument output failed: %v\n%s", err, immediateExecution)
	}
	if string(immediateExecution) != "PASS\n" {
		t.Fatalf("custom backend immediate argument output = %q, want PASS", immediateExecution)
	}
	if loaded := loadSeeded(first.prepared.Encoded, "example/example64"); !loaded.Ok {
		t.Fatalf("could not reload exported derived backend: %#v", loaded.Diagnostic)
	}
	for _, test := range []struct {
		name   string
		mutate func(*rtgb.Artifact)
	}{
		{"protocol", func(artifact *rtgb.Artifact) { artifact.Protocol++ }},
		{"unit", func(artifact *rtgb.Artifact) { artifact.Unit++ }},
		{"descriptor", func(artifact *rtgb.Artifact) { artifact.Descriptor.Version++ }},
		{"optimization", func(artifact *rtgb.Artifact) { artifact.Optimization++ }},
		{"generator", func(artifact *rtgb.Artifact) { artifact.Generator++ }},
		{"kernel", func(artifact *rtgb.Artifact) { artifact.Kernel++ }},
		{"host", func(artifact *rtgb.Artifact) { artifact.Host += "-wrong" }},
	} {
		artifact := first.prepared.Artifact
		test.mutate(&artifact)
		encoded, ok := rtgb.Encode(artifact)
		if !ok {
			t.Fatal("could not encode incompatible artifact")
		}
		if loaded := loadSeeded(encoded, "example/example64"); loaded.Ok {
			t.Fatalf("accepted artifact with incompatible %s identity", test.name)
		}
	}

	hostEmitters, err := filepath.Glob(filepath.Join(cache, "host-emitter-*.rtgb"))
	if err != nil || len(hostEmitters) != 1 {
		t.Fatalf("cold bootstrap host emitter cache = %q, error %v", hostEmitters, err)
	}
	if err = os.Rename(hostEmitters[0], hostEmitters[0]+".unavailable"); err != nil {
		t.Fatal(err)
	}
	second := NewSeeded(definition, stdRoot, cache)
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
	if second.prepared.ExecutablePath != first.prepared.ExecutablePath {
		t.Fatalf("warm compiler path = %q, want %q", second.prepared.ExecutablePath, first.prepared.ExecutablePath)
	}
	hostEmitters, err = filepath.Glob(filepath.Join(cache, "host-emitter-*.rtgb"))
	if err != nil || len(hostEmitters) != 0 {
		t.Fatalf("warm compiler cache unexpectedly rebuilt the VM host emitter: %q, error %v", hostEmitters, err)
	}
}

func TestNativeCompilerCacheRejectsAndReplacesCorruptExecutable(t *testing.T) {
	resolved := backendbuiltin.Resolve("linux/amd64")
	generated := rtg.GeneratePreparedBackend(resolved, "linux/amd64")
	if !generated.Ok {
		t.Fatalf("generate cache fixture descriptor: %#v", generated.Diagnostics)
	}
	artifact := seededArtifact(generated.Descriptor, "linux/amd64", []byte("\x7fELFcompiler-one"))
	directory := t.TempDir()
	key := "compiler-cache-recovery"
	stored, err := storeNativeCompilerCache(directory, key, artifact)
	if err != nil || !trustedCachedExecutable(stored.ExecutablePath) {
		t.Fatalf("store native compiler: %v, %#v", err, stored)
	}
	loaded, found := loadNativeCompilerCache(directory, key, generated.Descriptor, "linux/amd64")
	if !found || !loaded.CacheHit || !bytes.Equal(loaded.Artifact.Payload, artifact.Payload) {
		t.Fatalf("load native compiler = %#v, found %v", loaded, found)
	}
	if err = os.WriteFile(stored.ExecutablePath, []byte("corrupt compiler"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, found = loadNativeCompilerCache(directory, key, generated.Descriptor, "linux/amd64"); found {
		t.Fatal("cache accepted an executable that no longer matched its committed digest")
	}
	artifact.Payload = []byte("\x7fELFcompiler-two")
	if _, err = storeNativeCompilerCache(directory, key, artifact); err != nil {
		t.Fatalf("replace corrupt native compiler: %v", err)
	}
	loaded, found = loadNativeCompilerCache(directory, key, generated.Descriptor, "linux/amd64")
	if !found || !bytes.Equal(loaded.Artifact.Payload, artifact.Payload) {
		t.Fatal("replacement native compiler was not published atomically")
	}
}

func TestLinuxAmd64BuiltInProjectionMatchesPreparedDefinitionBehavior(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	stdRoot := filepath.Join(root, "std")
	source := filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go")
	builtInArgs := []string{"-t", "linux/amd64", "-s", "-o", "app", source}
	builtIn := driver.CompileFromFS(
		builtInArgs, root, stdRoot, driver.OSFS{}, backendcompiled.Backend{})
	if !builtIn.Ok {
		t.Fatalf("built-in linux/amd64 compile failed: %#v", builtIn.Diagnostic)
	}

	definition := copyNativeDefinition(t, root,
		"target linux/amd64 {", "target equivalence/linux64 {")
	preparedArgs := []string{
		"-backend", definition,
		"-t", "equivalence/linux64",
		"-s",
		"-o", "app",
		source,
	}
	preparedBackend := New(definition, filepath.Join(root, "backend"), stdRoot,
		t.TempDir(), backendcompiled.Backend{})
	prepared := driver.CompileFromFS(
		preparedArgs, root, stdRoot, driver.OSFS{}, preparedBackend)
	if !prepared.Ok {
		t.Fatalf("prepared equivalent linux/amd64 compile failed: %#v", prepared.Diagnostic)
	}

	outputs := make([][]byte, 0, 2)
	for _, image := range []struct {
		name   string
		binary []byte
	}{
		{name: "built-in", binary: builtIn.Binary},
		{name: "prepared", binary: prepared.Binary},
	} {
		parsed, err := elf.NewFile(bytes.NewReader(image.binary))
		if err != nil {
			t.Fatalf("parse %s linux/amd64 output: %v", image.name, err)
		}
		if parsed.Class != elf.ELFCLASS64 || parsed.Data != elf.ELFDATA2LSB ||
			parsed.Type != elf.ET_DYN || parsed.Machine != elf.EM_X86_64 {
			t.Fatalf("%s linux/amd64 ELF contract = class %v, data %v, type %v, machine %v",
				image.name, parsed.Class, parsed.Data, parsed.Type, parsed.Machine)
		}
		loadCount := 0
		hasReadExecute := false
		hasReadWrite := false
		entryInExecutableLoad := false
		for _, program := range parsed.Progs {
			if program.Type != elf.PT_LOAD {
				continue
			}
			loadCount++
			if program.Align != 0x1000 {
				t.Fatalf("%s linux/amd64 PT_LOAD alignment = %#x", image.name, program.Align)
			}
			switch program.Flags {
			case elf.PF_R | elf.PF_X:
				hasReadExecute = true
				entryInExecutableLoad = parsed.Entry >= program.Vaddr &&
					parsed.Entry < program.Vaddr+program.Memsz
			case elf.PF_R | elf.PF_W:
				hasReadWrite = true
			}
		}
		if loadCount != 2 || !hasReadExecute || !hasReadWrite || !entryInExecutableLoad {
			t.Fatalf("%s linux/amd64 load contract = count %d, rx %v, rw %v, entry-in-rx %v",
				image.name, loadCount, hasReadExecute, hasReadWrite, entryInExecutableLoad)
		}
		if err := parsed.Close(); err != nil {
			t.Fatalf("close %s linux/amd64 output: %v", image.name, err)
		}

		executable := filepath.Join(t.TempDir(), image.name+"-linux-amd64")
		if err := os.WriteFile(executable, image.binary, 0o700); err != nil {
			t.Fatal(err)
		}
		output, err := exec.Command(executable).CombinedOutput()
		if err != nil {
			t.Fatalf("execute %s linux/amd64 output: %v\n%s", image.name, err, output)
		}
		outputs = append(outputs, output)
	}
	if !bytes.Equal(outputs[0], outputs[1]) || string(outputs[0]) != "PASS\n" {
		t.Fatalf("linux/amd64 outputs: built-in %q, prepared %q, want PASS",
			outputs[0], outputs[1])
	}
}

func TestWindowsAmd64BuiltInProjectionMatchesPreparedDefinitionBehavior(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	stdRoot := filepath.Join(root, "std")
	source := filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go")
	builtIn := driver.CompileFromFS([]string{
		"-t", "windows/amd64", "-s", "-o", "app.exe", source,
	}, root, stdRoot, driver.OSFS{}, backendcompiled.Backend{})
	if !builtIn.Ok {
		t.Fatalf("built-in windows/amd64 compile failed: %#v", builtIn.Diagnostic)
	}

	definition := copyNativeDefinition(t, root,
		"target windows/amd64 {", "target equivalence/windows64 {")
	preparedBackend := New(definition, filepath.Join(root, "backend"), stdRoot,
		t.TempDir(), backendcompiled.Backend{})
	prepared := driver.CompileFromFS([]string{
		"-backend", definition, "-t", "equivalence/windows64", "-s",
		"-o", "app.exe", source,
	}, root, stdRoot, driver.OSFS{}, preparedBackend)
	if !prepared.Ok {
		t.Fatalf("prepared equivalent windows/amd64 compile failed: %#v", prepared.Diagnostic)
	}

	var expectedImports []string
	for _, image := range []struct {
		name string
		data []byte
	}{{"built-in", builtIn.Binary}, {"prepared", prepared.Binary}} {
		parsed, err := pe.NewFile(bytes.NewReader(image.data))
		if err != nil {
			t.Fatalf("parse %s windows/amd64 output: %v", image.name, err)
		}
		if parsed.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || len(parsed.Sections) != 2 {
			t.Fatalf("%s PE contract = machine %#x, sections %d",
				image.name, parsed.Machine, len(parsed.Sections))
		}
		header, ok := parsed.OptionalHeader.(*pe.OptionalHeader64)
		if !ok || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_CUI {
			t.Fatalf("%s PE optional header = %#v", image.name, parsed.OptionalHeader)
		}
		if parsed.Sections[0].Name != ".text" || parsed.Sections[1].Name != ".data" {
			t.Fatalf("%s PE sections = %q, %q", image.name,
				parsed.Sections[0].Name, parsed.Sections[1].Name)
		}
		imports, err := parsed.ImportedSymbols()
		if err != nil {
			t.Fatalf("read %s imports: %v", image.name, err)
		}
		if expectedImports == nil {
			expectedImports = imports
		} else if !equalStrings(expectedImports, imports) {
			t.Fatalf("prepared imports %q differ from built-in imports %q",
				imports, expectedImports)
		}
	}
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestCompiledInBootstrapUsesAArch64Definition(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
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
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		target  string
		custom  string
		machine []byte
	}{
		{"windows/amd64", "example/windows-amd64", []byte{0x64, 0x86}},
		{"windows/386", "example/windows-386", []byte{0x4c, 0x01}},
		{"windows/arm64", "example/windows-arm64", []byte{0x64, 0xaa}},
	}
	for _, test := range cases {
		t.Run(test.custom, func(t *testing.T) {
			t.Parallel()
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
	source := filepath.Join(
		root, "internal", "backendjit", "testdata", "kernel_runtime.go")
	builtIn := driver.CompileFromFS([]string{
		"-t", "linux/amd64",
		"-mode=kernel-module",
		"-o", "prepared.ko",
		source,
	}, root, filepath.Join(root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !builtIn.Ok {
		t.Fatalf("built-in kernel backend compile failed: %#v", builtIn.Diagnostic)
	}
	definition := copyNativeDefinition(t, root,
		"target linux-kernel/amd64 {", "target example/kernel-amd64 {")
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "example/kernel-amd64",
		"-mode=kernel-module",
		"-o", "prepared.ko",
		source,
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			t.TempDir(), backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("custom kernel backend compile failed: %#v", result.Diagnostic)
	}
	var expectedModuleInfo []byte
	var expectedImports []string
	for _, candidate := range []struct {
		name string
		data []byte
	}{{"built-in", builtIn.Binary}, {"prepared", result.Binary}} {
		parsed, parseErr := elf.NewFile(bytes.NewReader(candidate.data))
		if parseErr != nil {
			t.Fatalf("parse %s kernel module: %v", candidate.name, parseErr)
		}
		if parsed.Type != elf.ET_REL || parsed.Machine != elf.EM_X86_64 {
			t.Fatalf("%s kernel image type/machine = %v/%v",
				candidate.name, parsed.Type, parsed.Machine)
		}
		for _, name := range []string{
			".text", ".rela.text", ".modinfo", ".gnu.linkonce.this_module",
			".rela.gnu.linkonce.this_module",
		} {
			if parsed.Section(name) == nil {
				t.Fatalf("%s kernel image is missing %s", candidate.name, name)
			}
		}
		moduleInfo, dataErr := parsed.Section(".modinfo").Data()
		if dataErr != nil {
			t.Fatalf("read %s module information: %v", candidate.name, dataErr)
		}
		symbols, symbolErr := parsed.Symbols()
		if symbolErr != nil {
			t.Fatalf("read %s kernel symbols: %v", candidate.name, symbolErr)
		}
		var imports []string
		for i := 0; i < len(symbols); i++ {
			if symbols[i].Section == elf.SHN_UNDEF && symbols[i].Name != "" {
				imports = append(imports, symbols[i].Name)
			}
		}
		sort.Strings(imports)
		if expectedModuleInfo == nil {
			expectedModuleInfo = moduleInfo
			expectedImports = imports
		} else if !bytes.Equal(expectedModuleInfo, moduleInfo) ||
			!equalStrings(expectedImports, imports) {
			t.Fatalf("prepared kernel metadata/imports differ from built-in: %q/%q and %q/%q",
				moduleInfo, expectedModuleInfo, imports, expectedImports)
		}
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
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := copyNativeDefinition(t, root,
		"target darwin/arm64 {", "target example/darwin-arm64 {")
	prepared := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		t.TempDir(), backendcompiled.Backend{})
	result := driver.CompileFromFS([]string{
		"-backend", definition,
		"-t", "example/darwin-arm64",
		"-s",
		"-o", "app",
		filepath.Join(root, "backend", "tests", "rtg_aarch64_expression_stack.go"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		prepared)
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
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		executable := filepath.Join(t.TempDir(), "custom-aarch64-output")
		if err := os.WriteFile(executable, result.Binary, 0o700); err != nil {
			t.Fatal(err)
		}
		execution, err := exec.Command(executable).CombinedOutput()
		if err != nil {
			t.Fatalf("custom AArch64 output failed: %v\n%s", err, execution)
		}
		if string(execution) != "PASS\n" {
			t.Fatalf("custom AArch64 output = %q, want PASS", execution)
		}
	}
	for _, name := range []string{
		"rtg_aarch64_many_call_words.go",
		"rtg_aarch64_string_compare.go",
	} {
		t.Run(name, func(t *testing.T) {
			regression := driver.CompileFromFS([]string{
				"-backend", definition,
				"-t", "example/darwin-arm64",
				"-s",
				"-o", "app",
				filepath.Join(root, "backend", "tests", name),
			}, root, filepath.Join(root, "std"), driver.OSFS{}, prepared)
			if !regression.Ok {
				t.Fatalf("custom Darwin regression compile failed: %#v", regression.Diagnostic)
			}
			if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
				return
			}
			executable := filepath.Join(t.TempDir(), "custom-aarch64-regression")
			if err := os.WriteFile(executable, regression.Binary, 0o700); err != nil {
				t.Fatal(err)
			}
			execution, err := exec.Command(executable).CombinedOutput()
			if err != nil {
				t.Fatalf("custom AArch64 regression failed: %v\n%s", err, execution)
			}
			if string(execution) != "PASS\n" {
				t.Fatalf("custom AArch64 regression output = %q, want PASS", execution)
			}
		})
	}
}

func TestCompiledInBootstrapUses32BitDefinitions(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		target  string
		custom  string
		machine byte
	}{
		{"linux/386", "example/386", 3},
		{"linux/arm", "example/arm", 40},
	}
	for _, test := range cases {
		t.Run(test.custom, func(t *testing.T) {
			t.Parallel()
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
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
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
			t.Parallel()
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
	sourceRoot := filepath.Join(root, "backend", "definitions")
	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".rtg" &&
			entry.Name() != "wasm32.rtg" {
			names = append(names, entry.Name())
		}
	}
	files := make([]definitionFile, len(names))
	for i := 0; i < len(names); i++ {
		source, err := os.ReadFile(filepath.Join(sourceRoot, names[i]))
		if err != nil {
			t.Fatal(err)
		}
		files[i] = definitionFile{name: names[i], source: source}
	}
	entrypoint := ""
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
			if bytes.HasPrefix(old, []byte("target ")) {
				entrypoint = files[i].name
			}
			found = true
		}
		if !found {
			t.Fatalf("native definition replacement %q was not found", old)
		}
	}
	if entrypoint == "" {
		t.Fatal("copyNativeDefinition requires a target replacement")
	}
	destination := t.TempDir()
	for i := 0; i < len(files); i++ {
		if err := os.WriteFile(filepath.Join(destination, files[i].name),
			files[i].source, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(destination, entrypoint)
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
