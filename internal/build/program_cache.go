package build

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

const packageProgramCacheCapacity = 512

var packageProgramCacheUsed []bool
var packageProgramCachePackage []int
var packageProgramCacheGraphA []int
var packageProgramCacheGraphB []int
var packageProgramCacheSourceA []int
var packageProgramCacheSourceB []int
var packageProgramCachePathA []int
var packageProgramCachePathB []int
var packageProgramCacheData [][]byte
var packageProgramCacheNext int
var packageProgramCacheHits int
var packageProgramCacheMisses int

// InitializePackageProgramCache allocates cache headers before an embedded
// command takes its transient arena mark. Cached payloads themselves live in
// the persistent arena.
func InitializePackageProgramCache() {
	if len(packageProgramCacheUsed) != 0 {
		return
	}
	packageProgramCacheUsed = make([]bool, packageProgramCacheCapacity)
	packageProgramCachePackage = make([]int, packageProgramCacheCapacity)
	packageProgramCacheGraphA = make([]int, packageProgramCacheCapacity)
	packageProgramCacheGraphB = make([]int, packageProgramCacheCapacity)
	packageProgramCacheSourceA = make([]int, packageProgramCacheCapacity)
	packageProgramCacheSourceB = make([]int, packageProgramCacheCapacity)
	packageProgramCachePathA = make([]int, packageProgramCacheCapacity)
	packageProgramCachePathB = make([]int, packageProgramCacheCapacity)
	packageProgramCacheData = make([][]byte, packageProgramCacheCapacity)
}

func loadCachedPackageProgram(graph load.Graph, packageIndex int, contextA int, contextB int) (unit.Program, bool) {
	var empty unit.Program
	if packageIndex < 0 || packageIndex >= len(graph.Packages) {
		return empty, false
	}
	sourceA, sourceB := packageSourceHash(graph.Packages[packageIndex])
	pathA, pathB := packageCacheHashString(graph.Packages[packageIndex].Ref.ImportPath)
	for i := 0; i < packageProgramCacheCapacity; i++ {
		match := false
		if packageProgramCacheUsed[i] {
			match = packageProgramCachePackage[i] == packageIndex
		}
		if match {
			match = packageProgramCacheGraphA[i] == contextA
		}
		if match {
			match = packageProgramCacheGraphB[i] == contextB
		}
		if match {
			match = packageProgramCacheSourceA[i] == sourceA
		}
		if match {
			match = packageProgramCacheSourceB[i] == sourceB
		}
		if match {
			match = packageProgramCachePathA[i] == pathA
		}
		if match {
			match = packageProgramCachePathB[i] == pathB
		}
		if match {
			program, ok := unit.UnmarshalFrontendCache(packageProgramCacheData[i])
			if ok && program.ImportPath == graph.Packages[packageIndex].Ref.ImportPath && program.Package == graph.Packages[packageIndex].Name {
				packageProgramCacheHits++
				return program, true
			}
		}
	}
	packageProgramCacheMisses++
	return empty, false
}

func storeCachedPackageProgram(graph load.Graph, packageIndex int, contextA int, contextB int, program unit.Program) {
	if packageIndex < 0 || packageIndex >= len(graph.Packages) {
		return
	}
	data, ok := unit.MarshalFrontendCache(program)
	if !ok {
		return
	}
	sourceA, sourceB := packageSourceHash(graph.Packages[packageIndex])
	pathA, pathB := packageCacheHashString(graph.Packages[packageIndex].Ref.ImportPath)
	slot := -1
	for i := 0; i < packageProgramCacheCapacity; i++ {
		match := false
		if packageProgramCacheUsed[i] {
			match = packageProgramCachePackage[i] == packageIndex
		}
		if match {
			match = packageProgramCachePathA[i] == pathA
		}
		if match {
			match = packageProgramCachePathB[i] == pathB
		}
		if match {
			slot = i
			break
		}
	}
	if slot < 0 {
		slot = packageProgramCacheNext
		packageProgramCacheNext++
		if packageProgramCacheNext == packageProgramCacheCapacity {
			packageProgramCacheNext = 0
		}
	}
	if cap(packageProgramCacheData[slot]) == 0 {
		packageProgramCacheData[slot] = arena.PersistBytes(data)
	} else if len(data) <= cap(packageProgramCacheData[slot]) {
		packageProgramCacheData[slot] = packageProgramCacheData[slot][:len(data)]
		copy(packageProgramCacheData[slot], data)
	} else {
		// Persistent arena allocations cannot be individually released. Keep the
		// existing slot instead of stranding its buffer and growing without bound;
		// this package will simply be rebuilt until its payload fits this slot again.
		return
	}
	packageProgramCacheUsed[slot] = true
	packageProgramCachePackage[slot] = packageIndex
	packageProgramCacheGraphA[slot] = contextA
	packageProgramCacheGraphB[slot] = contextB
	packageProgramCacheSourceA[slot] = sourceA
	packageProgramCacheSourceB[slot] = sourceB
	packageProgramCachePathA[slot] = pathA
	packageProgramCachePathB[slot] = pathB
}

func packageGraphHash(graph load.Graph) (int, int) {
	a, b := 17, 29
	a, b = packageCacheHashMix(a, b, graph.Root)
	for i := 0; i < len(graph.Packages); i++ {
		pkg := graph.Packages[i]
		a, b = packageCacheHashMix(a, b, pkg.Ref.ImportPath)
		a, b = packageCacheHashMix(a, b, pkg.Name)
		for j := 0; j < len(pkg.Imports); j++ {
			a, b = packageCacheHashMix(a, b, pkg.Imports[j].ImportPath)
		}
		a = packageCacheHashInt(a, len(pkg.Imports))
		b = packageCacheHashIntB(b, len(pkg.Imports))
	}
	return a, b
}

func packageContextHashes(graph load.Graph) ([]int, []int) {
	keysA := make([]int, len(graph.Packages))
	keysB := make([]int, len(graph.Packages))
	for i := 0; i < len(graph.Packages); i++ {
		pkg := graph.Packages[i]
		a, b := packageSourceHash(pkg)
		a, b = packageCacheHashMix(a, b, pkg.Ref.ImportPath)
		a, b = packageCacheHashMix(a, b, pkg.Name)
		for j := 0; j < len(pkg.Imports); j++ {
			path := pkg.Imports[j].ImportPath
			a, b = packageCacheHashMix(a, b, path)
			dependency := -1
			for k := 0; k < i; k++ {
				if graph.Packages[k].Ref.ImportPath == path {
					dependency = k
					break
				}
			}
			if dependency >= 0 {
				a = packageCacheHashInt(a, keysA[dependency])
				b = packageCacheHashIntB(b, keysB[dependency])
			}
		}
		keysA[i] = packageCacheHashInt(a, len(pkg.Imports))
		keysB[i] = packageCacheHashIntB(b, len(pkg.Imports))
	}
	return keysA, keysB
}

func packageSourceHash(pkg load.Package) (int, int) {
	a, b := 37, 53
	for i := 0; i < len(pkg.Files); i++ {
		a, b = packageCacheHashMix(a, b, pkg.Files[i].Path)
		for j := 0; j < len(pkg.Files[i].Src); j++ {
			a = packageCacheHashInt(a, int(pkg.Files[i].Src[j]))
			b = packageCacheHashIntB(b, int(pkg.Files[i].Src[j]))
		}
		a = packageCacheHashInt(a, len(pkg.Files[i].Src))
		b = packageCacheHashIntB(b, len(pkg.Files[i].Src))
	}
	return a, b
}

func packageCacheHashString(value string) (int, int) {
	a, b := 71, 89
	return packageCacheHashMix(a, b, value)
}

func packageCacheHashMix(a int, b int, value string) (int, int) {
	for i := 0; i < len(value); i++ {
		a = packageCacheHashInt(a, int(value[i]))
		b = packageCacheHashIntB(b, int(value[i]))
	}
	a = packageCacheHashInt(a, len(value))
	b = packageCacheHashIntB(b, len(value))
	return a, b
}

func packageCacheHashInt(hash int, value int) int {
	return hash*131 + value + 1
}

func packageCacheHashIntB(hash int, value int) int {
	return hash*257 + value + 3
}
