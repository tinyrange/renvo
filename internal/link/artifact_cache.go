package link

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/unit"
)

const packageArtifactCacheCapacity = 512

var packageArtifactCacheUsed []bool
var packageArtifactCachePackage []int
var packageArtifactCacheGraphA []int
var packageArtifactCacheGraphB []int
var packageArtifactCacheSourceA []int
var packageArtifactCacheSourceB []int
var packageArtifactCacheContextA []int
var packageArtifactCacheContextB []int
var packageArtifactCachePathA []int
var packageArtifactCachePathB []int
var packageArtifactCacheData [][]byte
var packageArtifactCacheNext int
var packageArtifactCacheHits int
var packageArtifactCacheMisses int

// InitializePackageArtifactCache creates cache headers before the embedded
// compiler records its transient arena mark. Payloads use persistent storage.
func InitializePackageArtifactCache() {
	if len(packageArtifactCacheUsed) != 0 {
		return
	}
	packageArtifactCacheUsed = make([]bool, packageArtifactCacheCapacity)
	packageArtifactCachePackage = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheGraphA = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheGraphB = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheSourceA = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheSourceB = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheContextA = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheContextB = make([]int, packageArtifactCacheCapacity)
	packageArtifactCachePathA = make([]int, packageArtifactCacheCapacity)
	packageArtifactCachePathB = make([]int, packageArtifactCacheCapacity)
	packageArtifactCacheData = make([][]byte, packageArtifactCacheCapacity)
}

func loadPackageArtifact(pkg build.PackageUnit, packageIndex int, contextA int, contextB int) (unit.Program, bool) {
	var empty unit.Program
	pathA, pathB := incrementalArtifactHashString(307, 401, pkg.ImportPath)
	for i := 0; i < packageArtifactCacheCapacity; i++ {
		match := packageArtifactCacheUsed[i] && packageArtifactCachePackage[i] == packageIndex
		if match {
			match = packageArtifactCacheGraphA[i] == pkg.GraphKeyA && packageArtifactCacheGraphB[i] == pkg.GraphKeyB
		}
		if match {
			match = packageArtifactCacheSourceA[i] == pkg.SourceKeyA && packageArtifactCacheSourceB[i] == pkg.SourceKeyB
		}
		if match {
			match = packageArtifactCacheContextA[i] == contextA && packageArtifactCacheContextB[i] == contextB
		}
		if match {
			match = packageArtifactCachePathA[i] == pathA && packageArtifactCachePathB[i] == pathB
		}
		if match {
			artifact, ok := unit.UnmarshalFrontendCache(packageArtifactCacheData[i])
			if ok && artifact.ImportPath == pkg.ImportPath && artifact.Package == pkg.Name {
				packageArtifactCacheHits++
				return artifact, true
			}
		}
	}
	packageArtifactCacheMisses++
	return empty, false
}

func storePackageArtifact(pkg build.PackageUnit, packageIndex int, contextA int, contextB int, artifact unit.Program) {
	data, ok := unit.MarshalFrontendCache(artifact)
	if !ok {
		return
	}
	pathA, pathB := incrementalArtifactHashString(307, 401, pkg.ImportPath)
	slot := -1
	for i := 0; i < packageArtifactCacheCapacity; i++ {
		match := packageArtifactCacheUsed[i] && packageArtifactCachePackage[i] == packageIndex
		if match {
			match = packageArtifactCachePathA[i] == pathA && packageArtifactCachePathB[i] == pathB
		}
		if match {
			slot = i
			break
		}
	}
	if slot < 0 {
		slot = packageArtifactCacheNext
		packageArtifactCacheNext++
		if packageArtifactCacheNext == packageArtifactCacheCapacity {
			packageArtifactCacheNext = 0
		}
	}
	if cap(packageArtifactCacheData[slot]) == 0 {
		packageArtifactCacheData[slot] = arena.PersistBytes(data)
	} else if len(data) <= cap(packageArtifactCacheData[slot]) {
		packageArtifactCacheData[slot] = packageArtifactCacheData[slot][:len(data)]
		copy(packageArtifactCacheData[slot], data)
	} else {
		return
	}
	packageArtifactCacheUsed[slot] = true
	packageArtifactCachePackage[slot] = packageIndex
	packageArtifactCacheGraphA[slot] = pkg.GraphKeyA
	packageArtifactCacheGraphB[slot] = pkg.GraphKeyB
	packageArtifactCacheSourceA[slot] = pkg.SourceKeyA
	packageArtifactCacheSourceB[slot] = pkg.SourceKeyB
	packageArtifactCacheContextA[slot] = contextA
	packageArtifactCacheContextB[slot] = contextB
	packageArtifactCachePathA[slot] = pathA
	packageArtifactCachePathB[slot] = pathB
}
