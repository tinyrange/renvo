//go:build renvo && darwin && arm64

package driver

import "renvo.dev/internal/linkedimage"

const renvoRunJITStackSize = 8 * 1024 * 1024

const (
	renvoRunDarwinProtRead   = 1
	renvoRunDarwinProtWrite  = 2
	renvoRunDarwinProtExec   = 4
	renvoRunDarwinMapPrivate = 2
	renvoRunDarwinMapFixed   = 0x10
	renvoRunDarwinMapJIT     = 0x800
	renvoRunDarwinMapAnon    = 0x1000
	renvoRunDarwinRTLDNow    = 2
)

// renvo:linkstatic /usr/lib/libSystem.B.dylib,mmap
func renvoRunDarwinMmap(address int, size int, protection int, flags int, fd int, offset int) int {
	return -1
}

// renvo:linkstatic /usr/lib/libSystem.B.dylib,mprotect
func renvoRunDarwinMprotect(address int, size int, protection int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,munmap
func renvoRunDarwinMunmap(address int, size int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,dlopen
func renvoRunDarwinDlopen(path *byte, mode int) int { return 0 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,dlsym
func renvoRunDarwinDlsym(handle int, name *byte) int { return 0 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,dlclose
func renvoRunDarwinDlclose(handle int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,sys_icache_invalidate
func renvoRunDarwinInvalidateInstructionCache(address int, size int) {}

// renvo:linkstatic /usr/lib/libSystem.B.dylib,pthread_jit_write_protect_np
func renvoRunDarwinJITWriteProtect(enabled int) {}

type renvoRunMapping struct {
	base int
	size int
}

type renvoRunPersistentSymbol struct {
	id      int
	address int
	size    int
}

// LinkedImageSession retains successful Mach-O generations so code pointers,
// slices, and interfaces from earlier submissions remain valid.
type LinkedImageSession struct {
	mappings       []renvoRunMapping
	symbols        []renvoRunPersistentSymbol
	libraryHandles []int
	stack          int
}

func (s *LinkedImageSession) Prepare() {
	if cap(s.mappings) == 0 {
		s.mappings = make([]renvoRunMapping, 0, 2048)
	}
	if cap(s.symbols) == 0 {
		s.symbols = make([]renvoRunPersistentSymbol, 0, 8192)
	}
	if cap(s.libraryHandles) == 0 {
		s.libraryHandles = make([]int, 0, 8192)
	}
}

func (s *LinkedImageSession) Run(native []byte, script string, args []string, env []string) int {
	if renvoRunJITCall == nil {
		return -1
	}
	entry, memorySize, segments, imports, libraries, ok := linkedimage.DarwinLayout(native)
	if !ok {
		return -1
	}
	linkSymbols, ok := linkedimage.PersistentSymbols(native, memorySize)
	if !ok {
		return -1
	}
	newSymbols := 0
	for i := 0; i < len(linkSymbols); i++ {
		if s.findSymbol(linkSymbols[i].ID) < 0 {
			newSymbols++
		}
	}
	if len(s.mappings) >= cap(s.mappings) || len(s.symbols)+newSymbols > cap(s.symbols) {
		return -1
	}
	if len(s.libraryHandles)+len(libraries) > cap(s.libraryHandles) {
		return -1
	}
	memorySize = renvoRunDarwinPageAlign(memorySize)
	// Reserve one contiguous address range so the image's relative references
	// remain valid, then replace each segment with its own mapping. Executable
	// pages use MAP_JIT as required on Apple silicon; data and the arena remain
	// ordinary writable mappings instead of becoming read-only when JIT write
	// protection is enabled.
	base := renvoRunDarwinMmap(
		0, memorySize, 0,
		renvoRunDarwinMapPrivate|renvoRunDarwinMapAnon,
		-1, 0,
	)
	if base == -1 || base == 0 {
		return -1
	}
	jitWritable := false
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if segment.MemorySize == 0 {
			continue
		}
		start := renvoRunDarwinPageFloor(segment.Address)
		end := renvoRunDarwinPageAlign(segment.Address + segment.MemorySize)
		protection := renvoRunDarwinProtRead | renvoRunDarwinProtWrite
		flags := renvoRunDarwinMapPrivate | renvoRunDarwinMapAnon | renvoRunDarwinMapFixed
		if segment.Permissions&1 != 0 {
			protection |= renvoRunDarwinProtExec
			flags |= renvoRunDarwinMapJIT
			if !jitWritable {
				renvoRunDarwinJITWriteProtect(0)
				jitWritable = true
			}
		}
		mapped := renvoRunDarwinMmap(base+start, end-start, protection, flags, -1, 0)
		if mapped != base+start {
			if jitWritable {
				renvoRunDarwinJITWriteProtect(1)
			}
			renvoRunDarwinMunmap(base, memorySize)
			return -1
		}
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		for at := 0; at < segment.FileSize; at++ {
			renvoRunStoreByte(base+segment.Address+at, native[segment.FileOffset+at])
		}
	}
	handles := make([]int, len(libraries))
	for i := 0; i < len(libraries); i++ {
		path := renvoRunDarwinCString(libraries[i])
		handles[i] = renvoRunDarwinDlopen(&path[0], renvoRunDarwinRTLDNow)
		if handles[i] == 0 {
			if jitWritable {
				renvoRunDarwinJITWriteProtect(1)
			}
			renvoRunDarwinMunmap(base, memorySize)
			return -1
		}
		s.libraryHandles = append(s.libraryHandles, handles[i])
	}
	for i := 0; i < len(imports); i++ {
		item := imports[i]
		handle := 0
		for j := 0; j < len(libraries); j++ {
			if libraries[j] == item.Library {
				handle = handles[j]
				break
			}
		}
		if handle == 0 {
			if jitWritable {
				renvoRunDarwinJITWriteProtect(1)
			}
			renvoRunDarwinMunmap(base, memorySize)
			return -1
		}
		name := item.Name
		if len(name) > 0 && name[0] == '_' {
			name = name[1:]
		}
		symbolName := renvoRunDarwinCString(name)
		address := renvoRunDarwinDlsym(handle, &symbolName[0])
		if address == 0 {
			if jitWritable {
				renvoRunDarwinJITWriteProtect(1)
			}
			renvoRunDarwinMunmap(base, memorySize)
			return -1
		}
		renvoRunDarwinStoreWord(base+item.Address, address)
	}
	for i := 0; i < len(linkSymbols); i++ {
		symbol := linkSymbols[i]
		previous := s.findSymbol(symbol.ID)
		if previous < 0 || s.symbols[previous].size != symbol.Size {
			continue
		}
		oldAddress := s.symbols[previous].address
		newAddress := base + symbol.Address
		for at := 0; at < symbol.Size; at++ {
			renvoRunStoreByte(newAddress+at, renvoRunLoadByte(oldAddress+at))
		}
		renvoRunStoreByte(base+symbol.RestoreAddress, 1)
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if segment.MemorySize == 0 {
			continue
		}
		// MAP_JIT pages keep their VM protection RWX; Apple silicon enforces
		// write-vs-execute access for those pages with the per-thread
		// pthread_jit_write_protect_np switch. Applying mprotect to a MAP_JIT
		// mapping is rejected on native macOS even when narrowing it to RX.
		if segment.Permissions&1 != 0 {
			continue
		}
		protection := 0
		if segment.Permissions&4 != 0 {
			protection |= renvoRunDarwinProtRead
		}
		if segment.Permissions&2 != 0 {
			protection |= renvoRunDarwinProtWrite
		}
		if segment.Permissions&1 != 0 {
			protection |= renvoRunDarwinProtExec
		}
		start := renvoRunDarwinPageFloor(segment.Address)
		end := renvoRunDarwinPageAlign(segment.Address + segment.MemorySize)
		if renvoRunDarwinMprotect(base+start, end-start, protection) != 0 {
			if jitWritable {
				renvoRunDarwinJITWriteProtect(1)
			}
			renvoRunDarwinMunmap(base, memorySize)
			return -1
		}
	}
	renvoRunDarwinInvalidateInstructionCache(base, memorySize)
	if jitWritable {
		renvoRunDarwinJITWriteProtect(1)
	}
	if s.stack == 0 {
		s.stack = renvoRunDarwinMmap(
			0, renvoRunJITStackSize,
			renvoRunDarwinProtRead|renvoRunDarwinProtWrite,
			renvoRunDarwinMapPrivate|renvoRunDarwinMapAnon,
			-1, 0,
		)
	}
	if s.stack == -1 || s.stack == 0 {
		renvoRunDarwinMunmap(base, memorySize)
		s.stack = 0
		return -1
	}
	programArgs := make([]string, 1, len(args)+1)
	programArgs[0] = script
	programArgs = append(programArgs, args...)
	argWords := renvoRunStringWords(programArgs)
	envWords := renvoRunStringWords(env)
	exitCode := renvoRunJITCall(
		base+entry,
		s.stack+renvoRunJITStackSize,
		renvoRunIntPointer(argWords),
		len(programArgs),
		renvoRunIntPointer(envWords),
		len(env),
	)
	if exitCode != 0 {
		renvoRunDarwinMunmap(base, memorySize)
		return exitCode
	}
	s.mappings = append(s.mappings, renvoRunMapping{base: base, size: memorySize})
	for i := 0; i < len(linkSymbols); i++ {
		symbol := linkSymbols[i]
		current := renvoRunPersistentSymbol{
			id: symbol.ID, address: base + symbol.Address, size: symbol.Size,
		}
		previous := s.findSymbol(symbol.ID)
		if previous >= 0 {
			s.symbols[previous] = current
		} else {
			s.symbols = append(s.symbols, current)
		}
	}
	return 0
}

func (s *LinkedImageSession) findSymbol(id int) int {
	for i := 0; i < len(s.symbols); i++ {
		if s.symbols[i].id == id {
			return i
		}
	}
	return -1
}

func (s *LinkedImageSession) Reset() {
	for i := 0; i < len(s.mappings); i++ {
		renvoRunDarwinMunmap(s.mappings[i].base, s.mappings[i].size)
	}
	if s.stack != 0 {
		renvoRunDarwinMunmap(s.stack, renvoRunJITStackSize)
	}
	for i := len(s.libraryHandles) - 1; i >= 0; i-- {
		renvoRunDarwinDlclose(s.libraryHandles[i])
	}
	s.mappings = nil
	s.symbols = nil
	s.libraryHandles = nil
	s.stack = 0
}

func renvoRunDarwinCString(value string) []byte {
	out := make([]byte, len(value)+1)
	for i := 0; i < len(value); i++ {
		out[i] = value[i]
	}
	return out
}

func renvoRunDarwinStoreWord(address int, value int) {
	for i := 0; i < 8; i++ {
		renvoRunStoreByte(address+i, byte(value>>(i*8)))
	}
}

func renvoRunDarwinPageAlign(value int) int {
	return (value + 16383) & -16384
}

func renvoRunDarwinPageFloor(value int) int {
	return value & -16384
}
