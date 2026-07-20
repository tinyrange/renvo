package main

import (
	"bytes"
	"testing"

	"renvo.dev/backend/unit"
)

func TestDarwinObjectCacheRelinksChangedFunctionWithStableObjects(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldStrip := renvoCompilerStripSymbols
	oldEntries := renvoObjectCacheEntries
	oldStorage := renvoObjectCacheStorage
	oldStorageUsed := renvoObjectCacheStorageUsed
	oldHits := renvoObjectCacheHits
	oldMisses := renvoObjectCacheMisses
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerStripSymbols = oldStrip
		renvoObjectCacheEntries = oldEntries
		renvoObjectCacheStorage = oldStorage
		renvoObjectCacheStorageUsed = oldStorageUsed
		renvoObjectCacheHits = oldHits
		renvoObjectCacheMisses = oldMisses
	})
	renvoObjectCacheEntries = nil
	renvoObjectCacheStorage = nil
	renvoObjectCacheStorageUsed = 0
	renvoObjectCacheHits = 0
	renvoObjectCacheMisses = 0
	renvoInitializeObjectCache()
	renvoSetTarget(renvoTargetDarwinArm64)
	renvoCompilerStripSymbols = true

	first := compileDarwinObjectCacheSource(t, []byte("package main\nfunc helper() int { return 40 }\nfunc stable() int { print(\"A\\x00B\"); return 2 }\nfunc appMain() int { print(\"A\\x00B\"); print(\"AAAA\"); return helper() + stable() }\n"))
	if !first.ok {
		t.Fatal("initial object-cache compilation failed")
	}
	changedSource := []byte("package main\nfunc helper() int { return 40 }\nfunc stable() int { print(\"A\\x00B\"); return 2 }\nfunc appMain() int { print(\"A\\x00B\"); print(\"BBBB\"); return helper() + stable() }\n")
	changed := compileDarwinObjectCacheSource(t, changedSource)
	if !changed.ok || renvoObjectCacheHits == 0 {
		t.Fatalf("incremental compilation failed or reused no objects: ok=%v hits=%d misses=%d", changed.ok, renvoObjectCacheHits, renvoObjectCacheMisses)
	}

	entries := renvoObjectCacheEntries
	renvoObjectCacheEntries = nil
	cold := compileDarwinObjectCacheSource(t, changedSource)
	renvoObjectCacheEntries = entries
	if !cold.ok || !bytes.Equal(changed.data, cold.data) {
		t.Fatal("incrementally linked Mach-O differs from a cold backend link")
	}
}

func TestLinuxAmd64UsesSharedRelocatableObjectLinker(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldStrip := renvoCompilerStripSymbols
	oldEntries := renvoObjectCacheEntries
	oldStorage := renvoObjectCacheStorage
	oldStorageUsed := renvoObjectCacheStorageUsed
	oldHits := renvoObjectCacheHits
	oldMisses := renvoObjectCacheMisses
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerStripSymbols = oldStrip
		renvoObjectCacheEntries = oldEntries
		renvoObjectCacheStorage = oldStorage
		renvoObjectCacheStorageUsed = oldStorageUsed
		renvoObjectCacheHits = oldHits
		renvoObjectCacheMisses = oldMisses
	})
	renvoObjectCacheEntries = nil
	renvoObjectCacheStorage = nil
	renvoObjectCacheStorageUsed = 0
	renvoObjectCacheHits = 0
	renvoObjectCacheMisses = 0
	renvoInitializeObjectCache()
	renvoSetTarget(renvoTargetLinuxAmd64)
	renvoCompilerStripSymbols = true

	first := compileAmd64ObjectCacheSource(t, []byte("package main\nfunc helper() int { return 40 }\nfunc stable() int { return 2 }\nfunc appMain() int { return helper() + stable() + 1 }\n"))
	if !first.ok {
		t.Fatal("initial amd64 object compilation failed")
	}
	changedSource := []byte("package main\nfunc helper() int { return 40 }\nfunc stable() int { return 2 }\nfunc appMain() int { return helper() + stable() + 2 }\n")
	changed := compileAmd64ObjectCacheSource(t, changedSource)
	if !changed.ok || renvoObjectCacheHits == 0 {
		t.Fatalf("amd64 shared object linker reused no fragments: ok=%v hits=%d misses=%d", changed.ok, renvoObjectCacheHits, renvoObjectCacheMisses)
	}
	entries := renvoObjectCacheEntries
	renvoObjectCacheEntries = nil
	cold := compileAmd64ObjectCacheSource(t, changedSource)
	renvoObjectCacheEntries = entries
	if !cold.ok || !bytes.Equal(changed.data, cold.data) {
		t.Fatal("amd64 relocatable-object link differs from a cold ELF link")
	}
}

func compileAmd64ObjectCacheSource(t *testing.T, source []byte) renvoCompileResult {
	t.Helper()
	program := renvoParseProgram(source)
	if !program.ok {
		t.Fatal("amd64 cache source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("amd64 cache metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetLinuxAmd64)
	return renvoTryCompileScalarProgramAmd64(&program, &meta)
}

func compileDarwinObjectCacheSource(t *testing.T, source []byte) renvoCompileResult {
	t.Helper()
	program := renvoParseProgram(source)
	if !program.ok {
		t.Fatal("test source did not parse")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	if !meta.ok {
		t.Fatal("test source metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetDarwinArm64)
	return renvoTryCompileScalarProgramAarch64(&program, &meta)
}

func TestDarwinCompileSessionYieldsAndMatchesSynchronousOutput(t *testing.T) {
	program := unitProgramFromSource(t, []byte(`package main

func fifth() int { return 5 }
func fourth() int { return fifth() + 4 }
func third() int { return fourth() + 3 }
func second() int { return third() + 2 }
func first() int { return second() + 1 }
func appMain() int { return first() }
`))
	data, err := unit.Marshal(program)
	if err != nil {
		t.Fatal(err)
	}
	options := RenvoCompileOptions{StripSymbols: true}
	session := RenvoBeginCompileSession(data, "darwin/arm64", "-", options)
	steps := 0
	for session.stage < 4 && !session.done {
		session.Step()
		steps++
	}
	if session.done || !session.result.ok {
		t.Fatal("resumable compilation failed")
	}
	if steps < 4 {
		t.Fatalf("resumable compilation completed in only %d yielding steps", steps)
	}
	syncProgram, isUnit, decoded := renvoDecodeUnitProgram(data)
	if !isUnit || !decoded {
		t.Fatal("synchronous unit decode failed")
	}
	renvoSetStripSymbols(true)
	renvoSetTarget(renvoTargetDarwinArm64)
	syncResult := renvoCompileParsedProgram(&syncProgram, renvoTargetDarwinArm64)
	if !syncResult.ok || !bytes.Equal(session.result.data, syncResult.data) {
		t.Fatal("resumable and synchronous compiler outputs differ")
	}
}

func TestDarwinPackageObjectSurvivesUnrelatedFunctionIndexChange(t *testing.T) {
	oldTarget := renvoTarget
	oldArch := renvoTargetArch
	oldOS := renvoTargetOS
	oldStrip := renvoCompilerStripSymbols
	oldEntries := renvoObjectCacheEntries
	oldStorage := renvoObjectCacheStorage
	oldStorageUsed := renvoObjectCacheStorageUsed
	oldHits := renvoObjectCacheHits
	oldMisses := renvoObjectCacheMisses
	t.Cleanup(func() {
		renvoTarget = oldTarget
		renvoTargetArch = oldArch
		renvoTargetOS = oldOS
		renvoCompilerStripSymbols = oldStrip
		renvoObjectCacheEntries = oldEntries
		renvoObjectCacheStorage = oldStorage
		renvoObjectCacheStorageUsed = oldStorageUsed
		renvoObjectCacheHits = oldHits
		renvoObjectCacheMisses = oldMisses
	})
	renvoObjectCacheEntries = nil
	renvoObjectCacheStorage = nil
	renvoObjectCacheStorageUsed = 0
	renvoObjectCacheHits = 0
	renvoObjectCacheMisses = 0
	renvoInitializeObjectCache()
	renvoSetTarget(renvoTargetDarwinArm64)
	renvoCompilerStripSymbols = true

	first := compileDarwinPackageObjectUnit(t, []byte("package main\nfunc dependencyValue() int { return 40 }\nfunc appMain() int { return dependencyValue() + 2 }\n"), 1)
	if !first.ok {
		t.Fatal("initial package-object compilation failed")
	}
	beforeHits := renvoObjectCacheHits
	changedSource := []byte("package main\nfunc dependencyValue() int { return 40 }\nfunc rootExtra() int { return 7 }\nfunc appMain() int { return dependencyValue() + 2 }\n")
	changed := compileDarwinPackageObjectUnit(t, changedSource, 2)
	if !changed.ok || renvoObjectCacheHits <= beforeHits {
		t.Fatalf("stable dependency object was not reused after root function indexes changed: ok=%v hits=%d->%d misses=%d", changed.ok, beforeHits, renvoObjectCacheHits, renvoObjectCacheMisses)
	}

	entries := renvoObjectCacheEntries
	renvoObjectCacheEntries = nil
	cold := compileDarwinPackageObjectUnit(t, changedSource, 2)
	renvoObjectCacheEntries = entries
	if !cold.ok || !bytes.Equal(changed.data, cold.data) {
		t.Fatal("package-object relink differs from a cold Mach-O link")
	}
}

func compileDarwinPackageObjectUnit(t *testing.T, source []byte, rootFuncStart int) renvoCompileResult {
	t.Helper()
	program := unitProgramFromSource(t, source)
	if len(program.Funcs) <= rootFuncStart {
		t.Fatal("test program has too few functions")
	}
	tokenCount := len(program.Tokens) / 8
	program.Packages = []unit.PackageInfo{
		{Name: "dependency", ImportPath: "example/dependency", GraphKeyA: 101, GraphKeyB: 103, SourceKeyA: 107, SourceKeyB: 109, TextStart: 0, TextEnd: len(program.Text), TokenStart: 0, TokenEnd: tokenCount, DeclStart: 0, DeclEnd: 0, FuncStart: 0, FuncEnd: 1},
		{Name: "main", ImportPath: "example/main", GraphKeyA: 211 + rootFuncStart, GraphKeyB: 223 + rootFuncStart, SourceKeyA: 227 + rootFuncStart, SourceKeyB: 229 + rootFuncStart, TextStart: 0, TextEnd: len(program.Text), TokenStart: 0, TokenEnd: tokenCount, DeclStart: 0, DeclEnd: 0, FuncStart: rootFuncStart, FuncEnd: len(program.Funcs)},
	}
	data, err := unit.Marshal(program)
	if err != nil {
		t.Fatalf("%v (text=%d tokens=%d decls=%d funcs=%d root=%d)", err, len(program.Text), tokenCount, len(program.Decls), len(program.Funcs), rootFuncStart)
	}
	decoded, isUnit, ok := renvoDecodeUnitProgram(data)
	if !isUnit || !ok {
		t.Fatal("package ownership unit did not decode")
	}
	var meta renvoMeta
	renvoBuildMetaInto(&decoded, &meta)
	if !meta.ok {
		t.Fatal("package ownership metadata failed")
	}
	meta.arenaSize = renvoDefaultArenaSize(renvoTargetDarwinArm64)
	return renvoTryCompileScalarProgramAarch64(&decoded, &meta)
}
