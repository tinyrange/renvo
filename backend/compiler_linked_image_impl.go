package main

const renvoImageHeaderSize = 20

// renvoCompileOutputData wraps a linked native artifact in the target-neutral
// RNVI transport when requested. The transport is intentionally independent of
// ELF, PE, Mach-O, and WebAssembly; host consumers expose those payloads as one
// linked-image model and may load them without routing through a format writer.
func renvoCompileOutputData(data []byte, target int) []byte {
	if renvoFixedTarget != 0 {
		return data
	}
	return renvoCompileOutputDataWithContext(renvoLegacyCompileContext(), data, target)
}

func renvoCompileOutputDataWithContext(context *renvoCompileContext, data []byte, target int) []byte {
	if !context.emitImage {
		return data
	}
	format := 0
	if len(data) >= 2 && data[0] == 'M' && data[1] == 'Z' {
		format = 2
	} else if len(data) >= 4 {
		if data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
			format = 1
		} else if data[0] == 0xcf && data[1] == 0xfa && data[2] == 0xed && data[3] == 0xfe {
			format = 3
		} else if data[0] == 0 && data[1] == 'a' && data[2] == 's' && data[3] == 'm' {
			format = 4
		} else if data[0] == 'R' && data[1] == 'N' && data[2] == 'V' && data[3] == 'B' {
			format = 5
		}
	}
	out := make([]byte, 0, renvoImageHeaderSize+len(data))
	out = append(out, 'R', 'N', 'V', 'I')
	out = renvoAppend16(out, 1)
	out = renvoAppend16(out, renvoImageHeaderSize)
	out = renvoAppend16(out, target)
	out = append(out, byte(format), 0)
	out = renvoAppend32(out, len(data))
	checkA := 1
	checkB := 0
	for i := 0; i < len(data); i++ {
		checkA = (checkA + int(data[i])) % 65521
		checkB = (checkB + checkA) % 65521
	}
	out = renvoAppend16(out, checkA)
	out = renvoAppend16(out, checkB)
	out = append(out, data...)
	return out
}
func renvoLinearPrepareReplGlobals(g *renvoLinearGen) {
	renvoNonNil(g)
	if renvoFixedTarget != 0 || !g.c.emitImage ||
		g.c.renvoTargetArch == renvoArchWasm32 {
		return
	}
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == renvoTokVar && renvoReplSlotID(g.prog.src, s.nameStart, s.nameEnd) >= 0 {
			g.replRestoreOffsets = make([]int, len(g.meta.globals))
			for j := 0; j < len(g.replRestoreOffsets); j++ {
				g.replRestoreOffsets[j] = -1
			}
			return
		}
	}
}

func renvoLinearAddReplGlobal(g *renvoLinearGen, index int, offset int, size int) {
	renvoNonNil(g)
	if len(g.replRestoreOffsets) == 0 {
		return
	}
	s := &g.meta.globals[index]
	id := renvoReplSlotID(g.prog.src, s.nameStart, s.nameEnd)
	if id < 0 {
		return
	}
	restoreOff := g.asm.bssSize
	g.asm.bssSize += 8
	g.replRestoreOffsets[index] = restoreOff
	g.asm.replSymbols = append(g.asm.replSymbols, renvoReplSymbol{
		id: id, offset: offset, size: size, restoreOff: restoreOff,
	})
}

// renvoAppendReplLinkTable appends metadata after the native file's loadable
// bytes. Each entry contains a stable symbol ID, its BSS address, its exact
// copy size, and the address of the initializer-suppression flag. The outer
// RNVI checksum protects the table together with the executable.
func renvoAppendReplLinkTable(out []byte, a *renvoAsm) []byte {
	renvoNonNil(a)
	if len(a.replSymbols) == 0 {
		return out
	}
	bss := renvoAsmBssOffset(a)
	for i := 0; i < len(a.replSymbols); i++ {
		s := a.replSymbols[i]
		out = renvoAppend32(out, s.id)
		out = renvoAppend32(out, bss+s.offset)
		out = renvoAppend32(out, s.size)
		out = renvoAppend32(out, bss+s.restoreOff)
	}
	out = renvoAppend32(out, len(a.replSymbols))
	out = append(out, 'R', 'P', 'L', '1')
	return out
}
