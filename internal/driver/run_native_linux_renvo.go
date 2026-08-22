//go:build renvo && linux

package driver

import "renvo.dev/internal/linkedimage"

const renvoRunJITStackSize = 8 * 1024 * 1024

var renvoRunLinuxSyscall func(int, int, int, int, int, int, int) int

func SetRunLinuxSyscall(handler func(int, int, int, int, int, int, int) int) {
	renvoRunLinuxSyscall = handler
}

type renvoRunMapping struct {
	base int
	size int
}

type renvoRunPersistentSymbol struct {
	id      int
	address int
	size    int
}

// LinkedImageSession owns a chain of loaded image generations. Old mappings
// deliberately remain live: migrated slices, interfaces, closures, and
// pointers may still address storage or code from an earlier generation.
type LinkedImageSession struct {
	mappings []renvoRunMapping
	symbols  []renvoRunPersistentSymbol
	stack    int
}

// Prepare reserves session bookkeeping before the caller takes a transient
// compiler-arena mark. Run never grows these slices, so resetting compiler
// scratch memory cannot invalidate the live linker state.
func (s *LinkedImageSession) Prepare() {
	if cap(s.mappings) == 0 {
		s.mappings = make([]renvoRunMapping, 0, 2048)
	}
	if cap(s.symbols) == 0 {
		s.symbols = make([]renvoRunPersistentSymbol, 0, 8192)
	}
}

// RunNativeLinkedImage maps a Linux image into the current process and enters
// it on an isolated stack.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	var session LinkedImageSession
	exitCode := session.run(native, script, args, env, false)
	session.Reset()
	return exitCode
}

// Run incrementally links, enters, and—on successful execution—commits a new
// image generation to this session.
func (s *LinkedImageSession) Run(native []byte, script string, args []string, env []string) int {
	return s.run(native, script, args, env, true)
}

func (s *LinkedImageSession) run(native []byte, script string, args []string, env []string, retain bool) int {
	if renvoRunLinuxSyscall == nil || renvoRunJITCall == nil {
		return -1
	}
	entry, memorySize, segments, ok := linkedimage.LinuxLayout(native)
	if !ok {
		return -1
	}
	var linkSymbols []linkedimage.PersistentSymbol
	newSymbols := 0
	if retain {
		linkSymbols, ok = linkedimage.PersistentSymbols(native, memorySize)
		if !ok {
			return -1
		}
		for i := 0; i < len(linkSymbols); i++ {
			if s.findSymbol(linkSymbols[i].ID) < 0 {
				newSymbols++
			}
		}
		if len(s.mappings) >= cap(s.mappings) || len(s.symbols)+newSymbols > cap(s.symbols) {
			return -1
		}
	}
	memorySize = renvoRunPageAlign(memorySize)
	base := renvoRunLinuxSyscall(renvoRunMmapSyscall(), 0, memorySize, 3, 34, -1, 0)
	if renvoRunSyscallFailed(base) {
		return -1
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		for at := 0; at < segment.FileSize; at++ {
			renvoRunStoreByte(base+segment.Address+at, native[segment.FileOffset+at])
		}
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
		// BSS is zero-filled, so setting the low byte is enough for every
		// supported little-endian Linux target.
		renvoRunStoreByte(base+symbol.RestoreAddress, 1)
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if segment.MemorySize == 0 {
			continue
		}
		protection := 0
		if segment.Permissions&4 != 0 {
			protection |= 1
		}
		if segment.Permissions&2 != 0 {
			protection |= 2
		}
		if segment.Permissions&1 != 0 {
			protection |= 4
		}
		start := renvoRunPageFloor(segment.Address)
		end := renvoRunPageAlign(segment.Address + segment.MemorySize)
		status := renvoRunLinuxSyscall(renvoRunMprotectSyscall(), base+start, end-start, protection, 0, 0, 0)
		if renvoRunSyscallFailed(status) {
			renvoRunLinuxSyscall(renvoRunMunmapSyscall(), base, memorySize, 0, 0, 0, 0)
			return -1
		}
	}
	if s.stack == 0 {
		s.stack = renvoRunLinuxSyscall(renvoRunMmapSyscall(), 0, renvoRunJITStackSize, 3, 34, -1, 0)
	}
	if renvoRunSyscallFailed(s.stack) || s.stack == 0 {
		renvoRunLinuxSyscall(renvoRunMunmapSyscall(), base, memorySize, 0, 0, 0, 0)
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
		renvoRunLinuxSyscall(renvoRunMunmapSyscall(), base, memorySize, 0, 0, 0, 0)
		return exitCode
	}
	if retain {
		s.mappings = append(s.mappings, renvoRunMapping{base: base, size: memorySize})
		for i := 0; i < len(linkSymbols); i++ {
			symbol := linkSymbols[i]
			current := renvoRunPersistentSymbol{id: symbol.ID, address: base + symbol.Address, size: symbol.Size}
			previous := s.findSymbol(symbol.ID)
			if previous >= 0 {
				s.symbols[previous] = current
			} else {
				s.symbols = append(s.symbols, current)
			}
		}
	} else {
		renvoRunLinuxSyscall(renvoRunMunmapSyscall(), base, memorySize, 0, 0, 0, 0)
	}
	return exitCode
}

func (s *LinkedImageSession) findSymbol(id int) int {
	for i := 0; i < len(s.symbols); i++ {
		if s.symbols[i].id == id {
			return i
		}
	}
	return -1
}

// Reset releases every retained generation and the shared call stack.
func (s *LinkedImageSession) Reset() {
	if renvoRunLinuxSyscall != nil {
		for i := 0; i < len(s.mappings); i++ {
			mapping := s.mappings[i]
			renvoRunLinuxSyscall(renvoRunMunmapSyscall(), mapping.base, mapping.size, 0, 0, 0, 0)
		}
		if s.stack != 0 {
			renvoRunLinuxSyscall(renvoRunMunmapSyscall(), s.stack, renvoRunJITStackSize, 0, 0, 0, 0)
		}
	}
	s.mappings = nil
	s.symbols = nil
	s.stack = 0
}

func renvoRunSyscallFailed(value int) bool {
	return value < 0 && value >= -4095
}

func renvoRunPageAlign(value int) int {
	return (value + 4095) & -4096
}

func renvoRunPageFloor(value int) int {
	return value & -4096
}
