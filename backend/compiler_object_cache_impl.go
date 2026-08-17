package main

const renvoObjectCacheCapacity = 1024
const renvoObjectCacheStorageSize = 2097152
const renvoObjectMagic = 0x524f424a
const renvoObjectStringReloc = 3
const renvoObjectRelocLocal = 0
const renvoObjectRelocFunction = 1
const renvoObjectRelocRuntime = 2

var renvoObjectCacheEntries []renvoObjectCacheEntry
var renvoObjectCacheStorage []byte
var renvoObjectCacheStorageUsed int
var renvoObjectCacheHits int
var renvoObjectCacheMisses int

func renvoInitializeObjectCache() {
	if len(renvoObjectCacheEntries) != 0 {
		return
	}
	renvoObjectCacheEntries = make([]renvoObjectCacheEntry, renvoObjectCacheCapacity)
}

func renvoObjectHashInt(a int, b int, value int) (int, int) {
	return a*131 + value + 1, b*257 + value + 3
}

func renvoObjectHashRange(a int, b int, src []byte, start int, end int) (int, int) {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if end < start {
		return renvoObjectHashInt(a, b, -1)
	}
	for i := start; i < end; i++ {
		a, b = renvoObjectHashInt(a, b, int(renvo_runtime_UnsafeByteAt(src, i)))
	}
	return renvoObjectHashInt(a, b, end-start)
}

func renvoObjectFunctionHash(g *renvoLinearGen, fnIndex int) (int, int) {
	object := g.object
	if object != nil && fnIndex >= 0 && fnIndex < len(object.funcIdentityA) && len(object.funcIdentityA) == len(g.meta.funcs) {
		return object.funcIdentityA[fnIndex], object.funcIdentityB[fnIndex]
	}
	return renvoObjectFunctionHashRaw(g, fnIndex)
}

func renvoObjectFunctionHashRaw(g *renvoLinearGen, fnIndex int) (int, int) {
	a, b := 947, 1237
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return renvoObjectHashInt(a, b, -1)
	}
	fn := g.meta.funcs[fnIndex]
	packageIndex := renvoObjectFunctionPackage(g, fnIndex)
	if packageIndex >= 0 {
		pkg := renvoProgramPackages(g.prog)[packageIndex]
		a, b = renvoObjectHashInt(a, b, pkg.graphKeyA)
		a, b = renvoObjectHashInt(a, b, pkg.graphKeyB)
		a, b = renvoObjectHashInt(a, b, pkg.sourceKeyA)
		a, b = renvoObjectHashInt(a, b, pkg.sourceKeyB)
	} else {
		a, b = renvoObjectHashInt(a, b, g.object.contextA)
		a, b = renvoObjectHashInt(a, b, g.object.contextB)
	}
	a, b = renvoObjectHashInt(a, b, g.object.contextA)
	a, b = renvoObjectHashInt(a, b, g.object.contextB)
	a, b = renvoObjectHashRange(a, b, g.prog.src, fn.nameStart, fn.nameEnd)
	if fn.declIndex >= 0 && fn.declIndex < len(g.prog.funcs) {
		decl := g.prog.funcs[fn.declIndex]
		if decl.receiverStart < decl.receiverEnd {
			start := int(renvoTokStart(g.prog, decl.receiverStart))
			end := int(renvoTokEnd(g.prog, decl.receiverEnd-1))
			a, b = renvoObjectHashRange(a, b, g.prog.src, start, end)
		}
	}
	if fn.nameStart == fn.nameEnd {
		position := fn.literalTok
		if position <= 0 {
			position = fn.bodyStart
		}
		if position >= 0 && position < renvoTokCount(g.prog) {
			textPosition := int(renvoTokStart(g.prog, position))
			if packageIndex >= 0 {
				textPosition -= renvoProgramPackages(g.prog)[packageIndex].textStart
			}
			a, b = renvoObjectHashInt(a, b, textPosition)
		}
	}
	return a & 2147483647, b & 2147483647
}

func renvoObjectInitializeFunctionIdentities(g *renvoLinearGen) {
	object := g.object
	count := len(g.meta.funcs)
	object.funcIdentityA = make([]int, count)
	object.funcIdentityB = make([]int, count)
	object.funcNext = make([]int, count)
	bucketCount := count*2 + 1
	if bucketCount < 17 {
		bucketCount = 17
	}
	object.funcBuckets = make([]int, bucketCount)
	for i := 0; i < bucketCount; i++ {
		object.funcBuckets[i] = -1
	}
	for i := 0; i < count; i++ {
		a, b := renvoObjectFunctionHashRaw(g, i)
		object.funcIdentityA[i] = a
		object.funcIdentityB[i] = b
		bucket := (a ^ b) % bucketCount
		object.funcNext[i] = object.funcBuckets[bucket]
		object.funcBuckets[bucket] = i
	}
}

func renvoObjectFunctionPackage(g *renvoLinearGen, fnIndex int) int {
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return -1
	}
	fn := g.meta.funcs[fnIndex]
	packages := renvoProgramPackages(g.prog)
	for i := 0; i < len(packages); i++ {
		pkg := packages[i]
		if fn.declIndex >= pkg.funcStart && fn.declIndex < pkg.funcEnd {
			return i
		}
		if fn.declIndex >= 0 && fn.declIndex < len(g.prog.funcs) {
			continue
		}
		if fn.nameStart >= pkg.textStart && fn.nameStart < pkg.textEnd {
			return i
		}
		if fn.bodyStart >= 0 && fn.bodyStart < renvoTokCount(g.prog) {
			position := int(renvoTokStart(g.prog, fn.bodyStart))
			if position >= pkg.textStart && position < pkg.textEnd {
				return i
			}
		}
	}
	return -1
}

func renvoObjectFindFunction(g *renvoLinearGen, identityA int, identityB int) int {
	object := g.object
	if object != nil && len(object.funcBuckets) > 0 {
		bucket := (identityA ^ identityB) % len(object.funcBuckets)
		found := -1
		for i := object.funcBuckets[bucket]; i >= 0; i = object.funcNext[i] {
			if object.funcIdentityA[i] == identityA && object.funcIdentityB[i] == identityB {
				if found >= 0 {
					return -1
				}
				found = i
			}
		}
		return found
	}
	found := -1
	for i := 0; i < len(g.meta.funcs); i++ {
		a, b := renvoObjectFunctionHash(g, i)
		if a == identityA && b == identityB {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

func renvoObjectFunctionBodyHash(g *renvoLinearGen, fnIndex int) (int, int) {
	a, b := 947, 1237
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return renvoObjectHashInt(a, b, -1)
	}
	fn := g.meta.funcs[fnIndex]
	start := int(renvoTokStart(g.prog, fn.bodyStart-1))
	end := int(renvoTokEnd(g.prog, fn.bodyEnd))
	return renvoObjectHashRange(a, b, g.prog.src, start, end)
}

func renvoObjectGeneratorStateHash(g *renvoLinearGen) (int, int) {
	a, b := 1597, 2017
	values := []int{
		g.threadStatePointerOff, g.mainThreadStateOff,
		g.stringHeapOff, g.stringHeapEndOff,
		g.stringHeapDataOff, g.stringHeapReady,
		g.printIntBufferOff, g.darwinEntryOff, g.fixedTargetValue, g.fixedTargetState,
		len(g.meta.types), len(g.meta.fields), len(g.meta.captures), len(g.asm.staticImports),
		len(g.asm.darwinImports), len(g.asm.darwinImportLabels), len(g.asm.darwinImportUsed),
	}
	labels := []int{
		g.runtimeFaultLabel, g.runtimeNonNilLabel, g.runtimeSecondaryLabel, g.runtimeBoundsLabel,
		g.runtimeByteIndexLabel, g.runtimeWordIndexLabel, g.runtimeWideIndexLabel,
		g.runtimeSliceBoundsLabel, g.divideCheckLabel, g.remainderCheckLabel, g.streqLabel,
		g.append8Label, g.append64Label, g.appendAddrLabel, g.appendBytesLabel, g.arenaAllocLabel,
		g.makeZeroLabel, g.winReadLabel, g.winWriteLabel, g.printIntLabel,
	}
	bools := []bool{
		g.streqEmitted, g.append8Emitted, g.append64Emitted, g.appendAddrEmitted, g.appendBytesEmitted,
		g.makeZeroEmitted, g.winReadEmitted, g.winWriteEmitted, g.printIntEmitted,
	}
	for i := 0; i < len(values); i++ {
		a, b = renvoObjectHashInt(a, b, values[i])
	}
	for i := 0; i < len(labels); i++ {
		label := labels[i]
		if label >= len(g.funcLabels) {
			label -= len(g.funcLabels)
		}
		a, b = renvoObjectHashInt(a, b, label)
	}
	for i := 0; i < len(bools); i++ {
		value := 0
		if bools[i] {
			value = 1
		}
		a, b = renvoObjectHashInt(a, b, value)
	}
	for i := 0; i < len(g.asm.staticImports); i++ {
		a, b = renvoObjectHashInt(a, b, i+1)
	}
	for i := 0; i < len(g.asm.darwinImports); i++ {
		imp := g.asm.darwinImports[i]
		label := imp.label
		if label >= len(g.funcLabels) {
			label -= len(g.funcLabels)
		}
		a, b = renvoObjectHashInt(a, b, label)
		if imp.used {
			a, b = renvoObjectHashInt(a, b, 1)
		}
	}
	for i := 0; i < len(g.asm.darwinImportLabels); i++ {
		label := g.asm.darwinImportLabels[i]
		if label >= len(g.funcLabels) {
			label -= len(g.funcLabels)
		}
		a, b = renvoObjectHashInt(a, b, label)
	}
	for i := 0; i < len(g.asm.darwinImportUsed); i++ {
		if g.asm.darwinImportUsed[i] {
			a, b = renvoObjectHashInt(a, b, i+1)
		} else {
			a, b = renvoObjectHashInt(a, b, 0)
		}
	}
	return a, b
}

func renvoObjectKey(g *renvoLinearGen, fnIndex int, stateA int, stateB int) (int, int) {
	a, b := 313, 733
	packageIndex := renvoObjectFunctionPackage(g, fnIndex)
	if packageIndex >= 0 {
		pkg := renvoProgramPackages(g.prog)[packageIndex]
		a, b = renvoObjectHashInt(a, b, pkg.graphKeyA)
		a, b = renvoObjectHashInt(a, b, pkg.graphKeyB)
		a, b = renvoObjectHashInt(a, b, pkg.sourceKeyA)
		a, b = renvoObjectHashInt(a, b, pkg.sourceKeyB)
	} else {
		a, b = renvoObjectHashInt(a, b, g.object.contextA)
		a, b = renvoObjectHashInt(a, b, g.object.contextB)
	}
	fnA, fnB := renvoObjectFunctionHash(g, fnIndex)
	bodyA, bodyB := renvoObjectFunctionBodyHash(g, fnIndex)
	a, b = renvoObjectHashInt(a, b, fnA)
	a, b = renvoObjectHashInt(a, b, fnB)
	a, b = renvoObjectHashInt(a, b, bodyA)
	a, b = renvoObjectHashInt(a, b, bodyB)
	a, b = renvoObjectHashInt(a, b, stateA)
	a, b = renvoObjectHashInt(a, b, stateB)
	a, b = renvoObjectHashInt(a, b, len(g.asm.code)&15)
	return a, b
}

func renvoObjectCacheEntryFor(fnA int, fnB int, keyA int, keyB int) int {
	if len(renvoObjectCacheEntries) == 0 {
		return -1
	}
	slot := (fnA ^ fnB) % len(renvoObjectCacheEntries)
	entry := renvoObjectCacheEntries[slot]
	if entry.used && entry.target == renvoTarget && entry.fnA == fnA && entry.fnB == fnB && entry.keyA == keyA && entry.keyB == keyB {
		return slot
	}
	return -1
}

type renvoObjectReader struct {
	data []byte
	pos  int
	ok   bool
}

func renvoObjectReadNext(r *renvoObjectReader) int {
	if !r.ok || r.pos < 0 || r.pos+4 > len(r.data) {
		r.ok = false
		return 0
	}
	value := int(r.data[r.pos]) | int(r.data[r.pos+1])<<8 | int(r.data[r.pos+2])<<16 | int(r.data[r.pos+3])<<24
	r.pos += 4
	return value
}

func renvoReplayFunctionObject(g *renvoLinearGen, fnIndex int, data []byte) bool {
	r := renvoObjectReader{data: data, ok: true}
	magic := renvoObjectReadNext(&r)
	if !r.ok || magic != renvoObjectMagic {
		return false
	}
	codeLen := renvoObjectReadNext(&r)
	labelCount := renvoObjectReadNext(&r)
	funcLabelRel := renvoObjectReadNext(&r)
	relocCount := renvoObjectReadNext(&r)
	absCount := renvoObjectReadNext(&r)
	queueCount := renvoObjectReadNext(&r)
	lastStoreEnd := renvoObjectReadNext(&r)
	lastStoreOff := renvoObjectReadNext(&r)
	lastLoad := renvoObjectReadNext(&r)
	dataLen := renvoObjectReadNext(&r)
	bssDelta := renvoObjectReadNext(&r)
	oldDataBase := renvoObjectReadNext(&r)
	oldBssBase := renvoObjectReadNext(&r)
	stringCount := renvoObjectReadNext(&r)
	if !r.ok || codeLen < 0 || labelCount < 0 || relocCount < 0 || absCount < 0 || queueCount < 0 || dataLen < 0 || bssDelta < 0 || stringCount < 0 || r.pos+codeLen+dataLen > len(data) {
		return false
	}
	a := &g.asm
	codeBase := len(a.code)
	labelBase := len(a.labelPos)
	dataBase := len(a.data)
	bssBase := a.bssSize
	a.code = append(a.code, data[r.pos:r.pos+codeLen]...)
	r.pos += codeLen
	a.data = append(a.data, data[r.pos:r.pos+dataLen]...)
	r.pos += dataLen
	a.bssSize += bssDelta
	for i := 0; i < stringCount; i++ {
		off := renvoObjectReadNext(&r)
		length := renvoObjectReadNext(&r)
		if !r.ok || off < 0 || length < 0 || off+length >= dataLen {
			return false
		}
		a.objectStrings.refs = append(a.objectStrings.refs, dataBase+off, length)
	}
	if fnIndex < 0 || fnIndex >= len(g.funcLabels) || funcLabelRel < 0 || funcLabelRel > codeLen {
		return false
	}
	a.labelPos[g.funcLabels[fnIndex]] = int32(codeBase + funcLabelRel)
	for i := 0; i < labelCount; i++ {
		set := renvoObjectReadNext(&r)
		position := renvoObjectReadNext(&r)
		if !r.ok || set < 0 || set > 1 || position < 0 || position > codeLen {
			return false
		}
		if set == 0 {
			a.labelPos = append(a.labelPos, -1)
		} else {
			a.labelPos = append(a.labelPos, int32(codeBase+position))
		}
	}
	for i := 0; i < relocCount; i++ {
		at := renvoObjectReadNext(&r)
		kind := renvoObjectReadNext(&r)
		valueA := renvoObjectReadNext(&r)
		label := -1
		if kind == renvoObjectRelocLocal {
			label = labelBase + valueA
			if valueA < 0 || valueA >= labelCount {
				return false
			}
		} else if kind == renvoObjectRelocFunction {
			valueB := renvoObjectReadNext(&r)
			callee := renvoObjectFindFunction(g, valueA, valueB)
			if callee < 0 || callee >= len(g.funcLabels) {
				return false
			}
			renvoLinearMarkFunc(g, callee)
			label = g.funcLabels[callee]
		} else if kind == renvoObjectRelocRuntime {
			label = len(g.funcLabels) + valueA
			if valueA < 0 || label < 0 || label >= labelBase {
				return false
			}
		} else {
			return false
		}
		if !r.ok || at < 0 || at >= codeLen || label < 0 || label >= len(a.labelPos) {
			return false
		}
		a.relocs = append(a.relocs, int32((codeBase+at)&2147483647), int32(label&2147483647))
	}
	for i := 0; i < absCount; i++ {
		at := renvoObjectReadNext(&r)
		kind := renvoObjectReadNext(&r)
		off := 0
		resolvedString := false
		if kind == renvoObjectStringReloc {
			length := renvoObjectReadNext(&r)
			if !r.ok || length < 0 || r.pos+length > len(data) {
				return false
			}
			off = renvoAddStringData(g, data[r.pos:r.pos+length])
			r.pos += length
			kind = 0
			resolvedString = true
		} else {
			off = renvoObjectReadNext(&r)
		}
		if !r.ok || at < 0 || at >= codeLen || off < 0 {
			return false
		}
		if kind == renvoAbsBssReloc && off >= oldBssBase {
			off = bssBase + off - oldBssBase
		} else if kind == 0 && !resolvedString && off >= oldDataBase {
			off = dataBase + off - oldDataBase
		}
		a.absRelocs = append(a.absRelocs, int32((codeBase+at)&2147483647), int32(off&2147483647), int32(kind&2147483647))
	}
	for i := 0; i < queueCount; i++ {
		identityA := renvoObjectReadNext(&r)
		identityB := renvoObjectReadNext(&r)
		callee := renvoObjectFindFunction(g, identityA, identityB)
		if !r.ok || callee < 0 || callee >= len(g.meta.funcs) {
			return false
		}
		renvoLinearMarkFunc(g, callee)
	}
	if r.pos != len(data) {
		return false
	}
	a.lastPrimaryStoreEnd = lastStoreEnd - 1
	a.lastPrimaryStoreOff = lastStoreOff
	a.lastPrimaryLoad = lastLoad
	return true
}

func renvoStoreFunctionObject(g *renvoLinearGen, fnIndex int, keyA int, keyB int, codeBase int, labelBase int, relocBase int, absBase int, queueBase int, dataBase int, bssBase int, stringBase int) {
	a := &g.asm
	funcLabel := g.funcLabels[fnIndex]
	if funcLabel < 0 || funcLabel >= len(a.labelPos) || renvoAsmLabelPosition(a, funcLabel) < 0 {
		return
	}
	for i := 0; i < labelBase; i++ {
		position := renvoAsmLabelPosition(a, i)
		if i != funcLabel && position >= codeBase {
			return
		}
	}
	codeLen := len(a.code) - codeBase
	labelCount := len(a.labelPos) - labelBase
	relocCount := (len(a.relocs) - relocBase) / 2
	absCount := (len(a.absRelocs) - absBase) / 3
	queueCount := len(g.funcQueue) - queueBase
	dataLen := len(a.data) - dataBase
	bssDelta := a.bssSize - bssBase
	stringCount := (len(a.objectStrings.refs) - stringBase) / 2
	out := make([]byte, 0, 60+codeLen+dataLen+stringCount*8+labelCount*8+relocCount*8+absCount*12+queueCount*4)
	out = renvoAppend32(out, renvoObjectMagic)
	out = renvoAppend32(out, codeLen)
	out = renvoAppend32(out, labelCount)
	out = renvoAppend32(out, renvoAsmLabelPosition(a, funcLabel)-codeBase)
	out = renvoAppend32(out, relocCount)
	out = renvoAppend32(out, absCount)
	out = renvoAppend32(out, queueCount)
	out = renvoAppend32(out, a.lastPrimaryStoreEnd+1)
	out = renvoAppend32(out, a.lastPrimaryStoreOff)
	out = renvoAppend32(out, a.lastPrimaryLoad)
	out = renvoAppend32(out, dataLen)
	out = renvoAppend32(out, bssDelta)
	out = renvoAppend32(out, dataBase)
	out = renvoAppend32(out, bssBase)
	out = renvoAppend32(out, stringCount)
	out = append(out, a.code[codeBase:]...)
	out = append(out, a.data[dataBase:]...)
	for i := stringBase; i < len(a.objectStrings.refs); i += 2 {
		off := a.objectStrings.refs[i] - dataBase
		length := a.objectStrings.refs[i+1]
		if off < 0 || length < 0 || off+length >= dataLen {
			return
		}
		out = renvoAppend32(out, off)
		out = renvoAppend32(out, length)
	}
	for i := labelBase; i < len(a.labelPos); i++ {
		set := 0
		position := 0
		labelPosition := renvoAsmLabelPosition(a, i)
		if labelPosition >= 0 {
			set = 1
			position = labelPosition - codeBase
			if position < 0 || position > codeLen {
				return
			}
		}
		out = renvoAppend32(out, set)
		out = renvoAppend32(out, position)
	}
	for i := relocBase; i+1 < len(a.relocs); i += 2 {
		at := int(renvo_runtime_UnsafeInt32At(a.relocs, i)) & 2147483647
		label := int(renvo_runtime_UnsafeInt32At(a.relocs, i+1)) & 2147483647
		out = renvoAppend32(out, at-codeBase)
		function := -1
		for j := 0; j < len(g.funcLabels); j++ {
			if g.funcLabels[j] == label {
				function = j
				break
			}
		}
		if function >= 0 {
			identityA, identityB := renvoObjectFunctionHash(g, function)
			out = renvoAppend32(out, renvoObjectRelocFunction)
			out = renvoAppend32(out, identityA)
			out = renvoAppend32(out, identityB)
		} else if label >= labelBase {
			out = renvoAppend32(out, renvoObjectRelocLocal)
			out = renvoAppend32(out, label-labelBase)
		} else if label >= len(g.funcLabels) {
			out = renvoAppend32(out, renvoObjectRelocRuntime)
			out = renvoAppend32(out, label-len(g.funcLabels))
		} else {
			return
		}
	}
	for i := absBase; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) & 2147483647
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1)) & 2147483647
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2)) & 2147483647
		out = renvoAppend32(out, at-codeBase)
		if kind == 0 && off >= g.object.dataBase && off < dataBase {
			length := -1
			for j := 0; j < len(a.objectStrings.refs); j += 2 {
				if a.objectStrings.refs[j] == off {
					length = a.objectStrings.refs[j+1]
					break
				}
			}
			if length < 0 || off+length >= len(a.data) {
				return
			}
			out = renvoAppend32(out, renvoObjectStringReloc)
			out = renvoAppend32(out, length)
			out = append(out, a.data[off:off+length]...)
		} else {
			out = renvoAppend32(out, kind)
			out = renvoAppend32(out, off)
		}
	}
	for i := queueBase; i < len(g.funcQueue); i++ {
		identityA, identityB := renvoObjectFunctionHash(g, g.funcQueue[i])
		out = renvoAppend32(out, identityA)
		out = renvoAppend32(out, identityB)
	}
	if len(renvoObjectCacheEntries) == 0 || fnIndex < 0 {
		return
	}
	fnA, fnB := renvoObjectFunctionHash(g, fnIndex)
	slot := (fnA ^ fnB) % len(renvoObjectCacheEntries)
	entry := &renvoObjectCacheEntries[slot]
	if cap(entry.data) >= len(out) {
		entry.data = entry.data[:len(out)]
		copy(entry.data, out)
	} else {
		if entry.used || renvoObjectCacheStorageUsed+len(out) > len(renvoObjectCacheStorage) {
			return
		}
		start := renvoObjectCacheStorageUsed
		renvoObjectCacheStorageUsed += len(out)
		entry.data = renvoObjectCacheStorage[start:renvoObjectCacheStorageUsed:renvoObjectCacheStorageUsed]
		copy(entry.data, out)
	}
	entry.used = true
	entry.target = g.c.renvoTarget
	entry.fnA = fnA
	entry.fnB = fnB
	entry.keyA = keyA
	entry.keyB = keyB
}

func renvoEmitScalarFunctionObjectCached(g *renvoLinearGen, fnIndex int) bool {
	if len(renvoObjectCacheEntries) == 0 || g.c.renvoTargetArch == renvoArchWasm32 || !g.c.stripSymbols {
		return renvoEmitScalarFunctionScratch(g, fnIndex)
	}
	// A direct-mapped cache smaller than the function graph cannot retain one
	// complete build and turns a cold compile into hashing and serialization
	// work followed by predictable conflicts. Keep the bounded cache useful
	// for interactive programs and compile larger graphs directly.
	if len(g.meta.funcs) > len(renvoObjectCacheEntries) {
		return renvoEmitScalarFunctionScratch(g, fnIndex)
	}
	if len(renvoObjectCacheStorage) == 0 {
		renvoObjectCacheStorage = make([]byte, renvoObjectCacheStorageSize)
		renvoObjectCacheStorageUsed = 0
	}
	if g.object == nil {
		g.object = &renvoObjectGenState{}
		g.object.contextA, g.object.contextB = renvoObjectContextHash(g)
		renvoObjectInitializeFunctionIdentities(g)
		g.object.dataBase = len(g.asm.data)
	}
	stateA, stateB := renvoObjectGeneratorStateHash(g)
	keyA, keyB := renvoObjectKey(g, fnIndex, stateA, stateB)
	fnA, fnB := renvoObjectFunctionHash(g, fnIndex)
	if slot := renvoObjectCacheEntryFor(fnA, fnB, keyA, keyB); slot >= 0 {
		if renvoReplayFunctionObject(g, fnIndex, renvoObjectCacheEntries[slot].data) {
			renvoObjectCacheHits++
			return true
		}
	}
	renvoObjectCacheMisses++
	a := &g.asm
	codeBase := len(a.code)
	labelBase := len(a.labelPos)
	relocBase := len(a.relocs)
	absBase := len(a.absRelocs)
	queueBase := len(g.funcQueue)
	dataBase := len(a.data)
	bssBase := a.bssSize
	stringBase := len(a.objectStrings.refs)
	if !renvoEmitScalarFunctionScratch(g, fnIndex) {
		return false
	}
	postA, postB := renvoObjectGeneratorStateHash(g)
	if stateA == postA && stateB == postB {
		renvoStoreFunctionObject(g, fnIndex, keyA, keyB, codeBase, labelBase, relocBase, absBase, queueBase, dataBase, bssBase, stringBase)
	}
	return true
}
func renvoEmitAllQueuedFunctionsCached(g *renvoLinearGen) bool {
	renvoNonNil(g)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		if !renvoEmitScalarFunctionObjectCached(g, g.funcQueue[queueIndex]) {
			return false
		}
	}
	return true
}
