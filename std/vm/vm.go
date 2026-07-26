// Package vm executes deterministic Renvo bytecode.
package vm

const (
	TrapNone = iota
	TrapInvalidLimits
	TrapInvalidProgram
	TrapStepLimit
	TrapMemoryLimit
	TrapInvalidInstruction
	TrapDivisionByZero
	TrapOutputLimit
)

type Limits struct {
	Steps  int
	Memory int
}

// Config supplies deterministic execution limits and isolated process inputs.
type Config struct {
	Limits Limits
	Args   []string
	Env    []string
	Stdin  []byte
	Files  []File
}

// File is an input or output in the VM's isolated in-memory filesystem.
type File struct {
	Name string
	Data []byte
	Mode int
}

// Result records execution output, resource usage, and trap diagnostics.
type Result struct {
	ExitCode   int
	Steps      int
	PeakMemory int
	Output     []byte
	Stderr     []byte
	Files      []File
	Trap       int
	TrapPC     int
	TrapOpcode int
}

const (
	headerSize    = 40
	routineSize   = 12
	routineFramed = 1
	version       = 1
	minAddress    = 256
	sysGetdents64 = 217
)

const (
	regRax = iota
	regRdx
	regRcx
	regRdi
	regRsi
	regR8
	regR9
	regR10
	regCount
)

const (
	opExit = 1 + iota
	opBuildArgsEnv
	opMovRegImm
	opMovRegReg
	opPushReg
	opPushImm
	opPopReg
	opLoadStack
	opStoreStack
	opLeaStack
	opLoadMem
	opStoreMem
	opLoadIndex
	opStoreIndex
	opAddRegReg
	opSubRegReg
	opMulRegReg
	opDivRegReg
	opModRegReg
	opAndRegReg
	opOrRegReg
	opXorRegReg
	opAndNotRegReg
	opShlRegReg
	opShrRegReg
	opAddRegImm
	opMulRegImm
	opIncReg
	opIncMem
	opDecMem
	opBoolNot
	opNegReg
	opCmpRegImm
	opCmpRegReg
	opSetCond
	opJmp
	opJz
	opJnz
	opJCond
	opCall
	opRet
	opSyscall
	opNop
	opShrUnsignedRegReg
)

const (
	condEq = iota
	condNe
	condLt
	condLe
	condGt
	condGe
)

type image struct {
	code     []byte
	data     []byte
	dataBase int
	bssBase  int
	bssSize  int
	routines []routine
}

type routine struct {
	pc        int
	frameSize int
	framed    bool
}

type callFrame struct {
	pc      int
	fp      int
	sp      int
	routine int
	drop    int
}

type descriptor struct {
	file   int
	offset int
	open   bool
	data   []byte
}

type machine struct {
	image     image
	limits    Limits
	memory    []byte
	regs      [regCount]int32
	pc        int
	currentPC int
	sp        int
	stackBase int
	fp        int
	routine   int
	flag      int32
	calls     []callFrame
	steps     int
	peak      int
	output    []byte
	stderr    []byte
	trap      int
	args      []string
	env       []string
	stdin     []byte
	stdinAt   int
	files     []File
	fds       []descriptor
}

// Run executes program with empty process inputs.
func Run(program []byte, limits Limits) Result {
	return RunConfig(program, Config{Limits: limits})
}

// RunConfig executes program with explicitly supplied deterministic inputs.
func RunConfig(program []byte, config Config) Result {
	limits := config.Limits
	if limits.Steps <= 0 || limits.Memory < minAddress {
		return Result{Trap: TrapInvalidLimits}
	}
	img, ok := decodeImage(program)
	if !ok {
		return Result{Trap: TrapInvalidProgram}
	}
	staticEnd := img.bssBase + img.bssSize
	if staticEnd < img.bssBase || staticEnd > limits.Memory {
		return Result{Trap: TrapMemoryLimit}
	}
	m := machine{
		image:     img,
		limits:    limits,
		memory:    make([]byte, limits.Memory),
		sp:        staticEnd,
		stackBase: staticEnd,
		fp:        limits.Memory,
		peak:      staticEnd,
		routine:   0,
		args:      config.Args,
		env:       config.Env,
		stdin:     config.Stdin,
		fds:       make([]descriptor, 3),
	}
	for i := 0; i < len(config.Files); i++ {
		file := File{Name: config.Files[i].Name, Mode: config.Files[i].Mode}
		file.Data = append(file.Data, config.Files[i].Data...)
		m.files = append(m.files, file)
	}
	copy(m.memory[img.dataBase:img.dataBase+len(img.data)], img.data)
	for m.trap == TrapNone {
		if !m.step() {
			break
		}
	}
	return Result{
		ExitCode:   int(m.regs[regRax]),
		Steps:      m.steps,
		PeakMemory: m.peak,
		Output:     m.output,
		Stderr:     m.stderr,
		Files:      m.resultFiles(),
		Trap:       m.trap,
		TrapPC:     m.currentPC,
		TrapOpcode: m.currentOpcode(),
	}
}

func decodeImage(program []byte) (image, bool) {
	var img image
	if len(program) < headerSize ||
		program[0] != 'R' || program[1] != 'N' || program[2] != 'V' || program[3] != 'B' ||
		read16(program, 4) != version || read16(program, 6) != headerSize {
		return img, false
	}
	codeSize := read32(program, 8)
	dataSize := read32(program, 12)
	img.dataBase = read32(program, 16)
	img.bssBase = read32(program, 20)
	img.bssSize = read32(program, 24)
	routineCount := read32(program, 28)
	wantChecksum := read32(program, 32)
	if codeSize <= 0 || dataSize < 0 || routineCount <= 0 ||
		img.dataBase < minAddress || img.bssBase < img.dataBase+dataSize || img.bssSize < 0 {
		return image{}, false
	}
	tableSize := routineCount * routineSize
	codeAt := headerSize + tableSize
	dataAt := codeAt + codeSize
	if tableSize < 0 || codeAt < headerSize || dataAt < codeAt ||
		dataAt+dataSize < dataAt || dataAt+dataSize != len(program) {
		return image{}, false
	}
	if checksum(program[headerSize:]) != wantChecksum {
		return image{}, false
	}
	img.routines = make([]routine, routineCount)
	for i := 0; i < routineCount; i++ {
		at := headerSize + i*routineSize
		flags := read32(program, at+8)
		img.routines[i] = routine{pc: read32(program, at), frameSize: read32(program, at+4), framed: flags&routineFramed != 0}
		if img.routines[i].pc < 0 || img.routines[i].pc >= codeSize ||
			img.routines[i].frameSize < 0 || img.routines[i].frameSize&3 != 0 ||
			flags != 0 && flags != routineFramed ||
			i == 0 && img.routines[i].pc != 0 ||
			i > 0 && img.routines[i-1].pc >= img.routines[i].pc {
			return image{}, false
		}
	}
	img.code = program[codeAt:dataAt]
	img.data = program[dataAt:]
	if !validateCode(img.code, img.routines) {
		return image{}, false
	}
	return img, true
}

func validateCode(code []byte, routines []routine) bool {
	starts := make([]bool, len(code)+1)
	pc := 0
	for pc < len(code) {
		starts[pc] = true
		next := nextInstruction(code, pc)
		if next <= pc || next > len(code) || !validOperands(code, pc) {
			return false
		}
		pc = next
	}
	if pc != len(code) {
		return false
	}
	starts[len(code)] = true
	for i := 0; i < len(routines); i++ {
		if !starts[routines[i].pc] {
			return false
		}
	}
	for pc = 0; pc < len(code); pc = nextInstruction(code, pc) {
		op := int(code[pc])
		target := -1
		if op == opJmp || op == opJz || op == opJnz || op == opCall {
			target = read32(code, pc+1)
		} else if op == opJCond {
			target = read32(code, pc+2)
		}
		if target >= 0 && (target >= len(code) || !starts[target]) {
			return false
		}
		if op == opCall && findRoutine(routines, target) < 0 {
			return false
		}
	}
	return true
}

func validOperands(code []byte, pc int) bool {
	op := int(code[pc])
	if op < opExit || op > opShrUnsignedRegReg {
		return false
	}
	if op == opMovRegImm || op == opLoadStack || op == opStoreStack || op == opLeaStack ||
		op == opAddRegImm || op == opMulRegImm || op == opCmpRegImm {
		return int(code[pc+1]) < regCount
	}
	if op == opMovRegReg || op >= opAddRegReg && op <= opShrRegReg ||
		op == opCmpRegReg || op == opShrUnsignedRegReg {
		return int(code[pc+1]) < regCount && int(code[pc+2]) < regCount
	}
	if op == opPushReg || op == opPopReg || op == opIncReg || op == opIncMem ||
		op == opDecMem || op == opBoolNot || op == opNegReg {
		return int(code[pc+1]) < regCount
	}
	if op == opSetCond {
		return int(code[pc+1]) <= condGe
	}
	if op == opJCond {
		return int(code[pc+1]) <= condGe
	}
	if op == opLoadMem || op == opStoreMem {
		return int(code[pc+1]) < regCount && int(code[pc+2]) < regCount && validSize(int(code[pc+7]))
	}
	if op == opLoadIndex || op == opStoreIndex {
		return int(code[pc+1]) < regCount && int(code[pc+2]) < regCount &&
			int(code[pc+3]) < regCount && validSize(int(code[pc+9]))
	}
	return true
}

func validSize(size int) bool {
	return size >= 1 && size <= 4
}

func nextInstruction(code []byte, pc int) int {
	if pc < 0 || pc >= len(code) {
		return -1
	}
	op := int(code[pc])
	size := 1
	if op == opMovRegImm || op == opLoadStack || op == opStoreStack || op == opLeaStack ||
		op == opAddRegImm || op == opMulRegImm || op == opCmpRegImm || op == opJCond {
		size = 6
	} else if op == opMovRegReg || op >= opAddRegReg && op <= opShrRegReg ||
		op == opCmpRegReg || op == opShrUnsignedRegReg {
		size = 3
	} else if op == opPushReg || op == opPopReg || op == opIncReg || op == opIncMem ||
		op == opDecMem || op == opBoolNot || op == opNegReg || op == opSetCond {
		size = 2
	} else if op == opPushImm || op == opJmp || op == opJz || op == opJnz {
		size = 5
	} else if op == opLoadMem || op == opStoreMem {
		size = 8
	} else if op == opLoadIndex || op == opStoreIndex {
		size = 10
	} else if op == opCall {
		size = 9
	} else if op == opBuildArgsEnv {
		size = 13
	}
	return pc + size
}

func (m *machine) step() bool {
	if m.pc < 0 || m.pc >= len(m.image.code) {
		m.trap = TrapInvalidInstruction
		return false
	}
	m.currentPC = m.pc
	if !m.charge(1) {
		return false
	}
	code := m.image.code
	pc := m.pc
	op := int(code[pc])
	next := nextInstruction(code, pc)
	m.pc = next
	if op == opExit {
		return false
	}
	if op == opBuildArgsEnv {
		args, ok := m.buildStrings(m.args)
		if !ok {
			return false
		}
		env, ok := m.buildStrings(m.env)
		if !ok {
			return false
		}
		m.regs[regRdi] = int32(args)
		m.regs[regRsi] = int32(len(m.args))
		m.regs[regRdx] = int32(len(m.args))
		m.regs[regRcx] = int32(env)
		m.regs[regR8] = int32(len(m.env))
		m.regs[regR9] = int32(len(m.env))
		m.stackBase = m.sp
		return true
	}
	if op == opMovRegImm {
		m.regs[int(code[pc+1])] = int32(read32(code, pc+2))
		return true
	}
	if op == opMovRegReg {
		m.regs[int(code[pc+1])] = m.regs[int(code[pc+2])]
		return true
	}
	if op == opPushReg {
		return m.push(m.regs[int(code[pc+1])])
	}
	if op == opPushImm {
		return m.push(int32(read32(code, pc+1)))
	}
	if op == opPopReg {
		value, ok := m.pop()
		if ok {
			m.regs[int(code[pc+1])] = value
		}
		return ok
	}
	if op == opLoadStack || op == opStoreStack || op == opLeaStack {
		reg := int(code[pc+1])
		addr := m.fp - int(int32(read32(code, pc+2)))
		if op == opLeaStack {
			if !m.address(addr, 0) {
				return false
			}
			m.regs[reg] = int32(addr)
			return true
		}
		if op == opLoadStack {
			value, ok := m.load(addr, 4)
			if ok {
				m.regs[reg] = int32(value)
			}
			return ok
		}
		return m.store(addr, 4, uint32(m.regs[reg]))
	}
	if op == opLoadMem || op == opStoreMem {
		reg := int(code[pc+1])
		base := int(code[pc+2])
		addr := int(m.regs[base]) + int(int32(read32(code, pc+3)))
		size := int(code[pc+7])
		if op == opLoadMem {
			value, ok := m.load(addr, size)
			if ok {
				m.regs[reg] = int32(value)
			}
			return ok
		}
		return m.store(addr, size, uint32(m.regs[reg]))
	}
	if op == opLoadIndex || op == opStoreIndex {
		reg := int(code[pc+1])
		base := int(code[pc+2])
		index := int(code[pc+3])
		scale := int(code[pc+4])
		addr := int(m.regs[base]) + int(m.regs[index])*scale + int(int32(read32(code, pc+5)))
		size := int(code[pc+9])
		if op == opLoadIndex {
			value, ok := m.load(addr, size)
			if ok {
				m.regs[reg] = int32(value)
			}
			return ok
		}
		return m.store(addr, size, uint32(m.regs[reg]))
	}
	if op >= opAddRegReg && op <= opShrRegReg || op == opShrUnsignedRegReg {
		return m.binary(op, int(code[pc+1]), int(code[pc+2]))
	}
	if op == opAddRegImm || op == opMulRegImm {
		reg := int(code[pc+1])
		imm := int32(read32(code, pc+2))
		if op == opAddRegImm {
			m.regs[reg] += imm
		} else {
			m.regs[reg] *= imm
		}
		return true
	}
	if op == opIncReg {
		m.regs[int(code[pc+1])]++
		return true
	}
	if op == opIncMem || op == opDecMem {
		addr := int(m.regs[int(code[pc+1])])
		value, ok := m.load(addr, 4)
		if !ok {
			return false
		}
		if op == opIncMem {
			value++
		} else {
			value--
		}
		return m.store(addr, 4, value)
	}
	if op == opBoolNot {
		reg := int(code[pc+1])
		if m.regs[reg] == 0 {
			m.regs[reg] = 1
		} else {
			m.regs[reg] = 0
		}
		return true
	}
	if op == opNegReg {
		reg := int(code[pc+1])
		m.regs[reg] = -m.regs[reg]
		return true
	}
	if op == opCmpRegImm {
		m.flag = m.regs[int(code[pc+1])] - int32(read32(code, pc+2))
		return true
	}
	if op == opCmpRegReg {
		m.flag = m.regs[int(code[pc+1])] - m.regs[int(code[pc+2])]
		return true
	}
	if op == opSetCond {
		if condition(m.flag, int(code[pc+1])) {
			m.regs[regRax] = 1
		} else {
			m.regs[regRax] = 0
		}
		return true
	}
	if op == opJmp {
		m.pc = read32(code, pc+1)
		return true
	}
	if op == opJz || op == opJnz {
		zero := m.flag == 0
		if op == opJz && zero || op == opJnz && !zero {
			m.pc = read32(code, pc+1)
		}
		return true
	}
	if op == opJCond {
		if condition(m.flag, int(code[pc+1])) {
			m.pc = read32(code, pc+2)
		}
		return true
	}
	if op == opCall {
		target := read32(code, pc+1)
		targetRoutine := findRoutine(m.image.routines, target)
		if targetRoutine < 0 {
			m.trap = TrapInvalidInstruction
			return false
		}
		frameSize := 0
		if m.image.routines[targetRoutine].framed {
			frameSize = m.image.routines[m.routine].frameSize
		}
		newFP := m.fp - frameSize
		if newFP < m.sp || newFP < 0 {
			m.trap = TrapMemoryLimit
			return false
		}
		wordCount := read32(code, pc+5)
		drop := 0
		if wordCount > 6 {
			drop = (wordCount - 6) * 4
		}
		m.calls = append(m.calls, callFrame{pc: next, fp: m.fp, sp: m.sp, routine: m.routine, drop: drop})
		m.fp = newFP
		m.routine = targetRoutine
		m.pc = target
		return m.updatePeak()
	}
	if op == opRet {
		if len(m.calls) == 0 {
			return false
		}
		frame := m.calls[len(m.calls)-1]
		m.calls = m.calls[:len(m.calls)-1]
		if frame.drop > frame.sp-m.stackBase {
			m.trap = TrapInvalidInstruction
			return false
		}
		m.sp = frame.sp - frame.drop
		m.pc = frame.pc
		m.fp = frame.fp
		m.routine = frame.routine
		return true
	}
	if op == opSyscall {
		return m.syscall()
	}
	if op == opNop {
		return true
	}
	m.trap = TrapInvalidInstruction
	return false
}

func (m *machine) currentOpcode() int {
	if m.currentPC < 0 || m.currentPC >= len(m.image.code) {
		return 0
	}
	return int(m.image.code[m.currentPC])
}

func (m *machine) binary(op int, dst int, src int) bool {
	left := m.regs[dst]
	right := m.regs[src]
	switch op {
	case opAddRegReg:
		m.regs[dst] = left + right
	case opSubRegReg:
		m.regs[dst] = left - right
	case opMulRegReg:
		m.regs[dst] = left * right
	case opDivRegReg:
		if right == 0 {
			m.trap = TrapDivisionByZero
			return false
		}
		if left == -2147483647-1 && right == -1 {
			m.regs[dst] = left
		} else {
			m.regs[dst] = left / right
		}
	case opModRegReg:
		if right == 0 {
			m.trap = TrapDivisionByZero
			return false
		}
		if left == -2147483647-1 && right == -1 {
			m.regs[dst] = 0
		} else {
			m.regs[dst] = left % right
		}
	case opAndRegReg:
		m.regs[dst] = left & right
	case opOrRegReg:
		m.regs[dst] = left | right
	case opXorRegReg:
		m.regs[dst] = left ^ right
	case opAndNotRegReg:
		m.regs[dst] = left &^ right
	case opShlRegReg:
		m.regs[dst] = left << (uint32(right) & 31)
	case opShrRegReg:
		m.regs[dst] = left >> (uint32(right) & 31)
	case opShrUnsignedRegReg:
		m.regs[dst] = int32(uint32(left) >> (uint32(right) & 31))
	default:
		m.trap = TrapInvalidInstruction
		return false
	}
	return true
}

func (m *machine) syscall() bool {
	number := int(m.regs[regRax])
	if number == 0 || number == 1 {
		fd := int(m.regs[regRdi])
		addr := int(m.regs[regRsi])
		size := int(m.regs[regRdx])
		if !m.buffer(addr, size) {
			return false
		}
		if number == 0 {
			return m.readAt(fd, addr, size, -1)
		}
		return m.writeAt(fd, addr, size, -1)
	}
	if number == 3 {
		fd := int(m.regs[regRdi])
		if fd < 3 || fd >= len(m.fds) || !m.fds[fd].open {
			m.regs[regRax] = -1
			return true
		}
		m.fds[fd].open = false
		m.regs[regRax] = 0
		return true
	}
	if number == 17 {
		return m.readAt(int(m.regs[regRdi]), int(m.regs[regRsi]), int(m.regs[regRdx]), int(m.regs[regR10]))
	}
	if number == 18 {
		return m.writeAt(int(m.regs[regRdi]), int(m.regs[regRsi]), int(m.regs[regRdx]), int(m.regs[regR10]))
	}
	if number == sysGetdents64 {
		fd := int(m.regs[regRdi])
		if fd < 3 || fd >= len(m.fds) || !m.fds[fd].open || m.fds[fd].file >= 0 {
			m.regs[regRax] = -1
			return true
		}
		return m.readAt(fd, int(m.regs[regRsi]), int(m.regs[regRdx]), -1)
	}
	if number == 2 {
		return m.open()
	}
	if number == 91 {
		fd := int(m.regs[regRdi])
		if fd < 3 || fd >= len(m.fds) || !m.fds[fd].open || m.fds[fd].file < 0 {
			m.regs[regRax] = -1
			return true
		}
		m.files[m.fds[fd].file].Mode = int(m.regs[regRsi])
		m.regs[regRax] = 0
		return true
	}
	m.regs[regRax] = -1
	return true
}

func (m *machine) open() bool {
	addr := int(m.regs[regRdi])
	size := int(m.regs[regRdx])
	flags := int(m.regs[regRsi])
	if !m.buffer(addr, size) {
		return false
	}
	if size > 0 && m.memory[addr+size-1] == 0 {
		size--
	}
	name := string(m.memory[addr : addr+size])
	file := -1
	for i := 0; i < len(m.files); i++ {
		if m.files[i].Name == name {
			file = i
			break
		}
	}
	const create = 64
	const exclusive = 128
	const truncate = 512
	if file < 0 {
		if data, ok := m.directoryData(name); ok {
			if flags&(create|truncate) != 0 {
				m.regs[regRax] = -1
				return true
			}
			fd := len(m.fds)
			m.fds = append(m.fds, descriptor{file: -1, open: true, data: data})
			m.regs[regRax] = int32(fd)
			return true
		}
	}
	if file >= 0 && flags&create != 0 && flags&exclusive != 0 {
		m.regs[regRax] = -1
		return true
	}
	if file < 0 {
		if flags&create == 0 {
			m.regs[regRax] = -1
			return true
		}
		m.files = append(m.files, File{Name: name, Mode: 493})
		file = len(m.files) - 1
	}
	if flags&truncate != 0 {
		m.files[file].Data = nil
	}
	fd := len(m.fds)
	m.fds = append(m.fds, descriptor{file: file, open: true})
	m.regs[regRax] = int32(fd)
	return true
}

func (m *machine) readAt(fd int, addr int, size int, offset int) bool {
	if !m.buffer(addr, size) {
		return false
	}
	var data []byte
	at := offset
	if fd == 0 {
		data = m.stdin
		if at < 0 {
			at = m.stdinAt
		}
	} else {
		if fd < 3 || fd >= len(m.fds) || !m.fds[fd].open {
			m.regs[regRax] = -1
			return true
		}
		if m.fds[fd].file < 0 {
			data = m.fds[fd].data
		} else {
			data = m.files[m.fds[fd].file].Data
		}
		if at < 0 {
			at = m.fds[fd].offset
		}
	}
	if at < 0 {
		m.regs[regRax] = -1
		return true
	}
	count := size
	if at > len(data) {
		count = 0
	} else if count > len(data)-at {
		count = len(data) - at
	}
	if !m.charge(count) {
		return false
	}
	copy(m.memory[addr:addr+count], data[at:at+count])
	if offset < 0 {
		if fd == 0 {
			m.stdinAt = at + count
		} else {
			m.fds[fd].offset = at + count
		}
	}
	m.regs[regRax] = int32(count)
	return true
}

func (m *machine) writeAt(fd int, addr int, size int, offset int) bool {
	if !m.buffer(addr, size) || !m.charge(size) {
		return false
	}
	if fd == 1 || fd == 2 {
		var output *[]byte
		if fd == 1 {
			output = &m.output
		} else {
			output = &m.stderr
		}
		if len(*output)+size > m.limits.Memory {
			m.trap = TrapOutputLimit
			return false
		}
		*output = append(*output, m.memory[addr:addr+size]...)
		m.regs[regRax] = int32(size)
		return true
	}
	if fd < 3 || fd >= len(m.fds) || !m.fds[fd].open || m.fds[fd].file < 0 {
		m.regs[regRax] = -1
		return true
	}
	at := offset
	if at < 0 {
		at = m.fds[fd].offset
	}
	if at < 0 || at > m.limits.Memory || size > m.limits.Memory-at {
		m.regs[regRax] = -1
		return true
	}
	file := &m.files[m.fds[fd].file]
	if at+size > len(file.Data) {
		next := make([]byte, at+size)
		copy(next, file.Data)
		file.Data = next
	}
	copy(file.Data[at:at+size], m.memory[addr:addr+size])
	if offset < 0 {
		m.fds[fd].offset = at + size
	}
	m.regs[regRax] = int32(size)
	return true
}

type directoryEntry struct {
	name  string
	isDir bool
}

func (m *machine) directoryData(name string) ([]byte, bool) {
	prefix := name
	if len(prefix) == 0 || prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	var entries []directoryEntry
	for i := 0; i < len(m.files); i++ {
		path := m.files[i].Name
		if len(path) <= len(prefix) || !stringPrefix(path, prefix) {
			continue
		}
		child := path[len(prefix):]
		isDir := false
		for j := 0; j < len(child); j++ {
			if child[j] == '/' {
				child = child[:j]
				isDir = true
				break
			}
		}
		if len(child) == 0 {
			continue
		}
		at := 0
		for at < len(entries) && entries[at].name < child {
			at++
		}
		if at < len(entries) && entries[at].name == child {
			if isDir {
				entries[at].isDir = true
			}
			continue
		}
		entries = append(entries, directoryEntry{})
		for j := len(entries) - 1; j > at; j-- {
			entries[j] = entries[j-1]
		}
		entries[at] = directoryEntry{name: child, isDir: isDir}
	}
	if len(entries) == 0 {
		return nil, false
	}
	var data []byte
	for i := 0; i < len(entries); i++ {
		recordSize := (20 + len(entries[i].name) + 7) &^ 7
		at := len(data)
		data = append(data, make([]byte, recordSize)...)
		data[at] = byte(i + 1)
		data[at+16] = byte(recordSize)
		data[at+17] = byte(recordSize >> 8)
		data[at+18] = 8
		if entries[i].isDir {
			data[at+18] = 4
		}
		for j := 0; j < len(entries[i].name); j++ {
			data[at+19+j] = entries[i].name[j]
		}
	}
	return data, true
}

func stringPrefix(text string, prefix string) bool {
	if len(text) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if text[i] != prefix[i] {
			return false
		}
	}
	return true
}

func (m *machine) resultFiles() []File {
	result := make([]File, len(m.files))
	for i := 0; i < len(m.files); i++ {
		result[i] = File{Name: m.files[i].Name, Mode: m.files[i].Mode}
		result[i].Data = append(result[i].Data, m.files[i].Data...)
	}
	return result
}

func (m *machine) push(value int32) bool {
	if !m.address(m.sp, 4) || m.sp+4 > m.fp {
		m.trap = TrapMemoryLimit
		return false
	}
	write32(m.memory, m.sp, int(value))
	m.sp += 4
	return m.updatePeak()
}

func (m *machine) buildStrings(values []string) (int, bool) {
	start := m.sp
	dataAt := align8(start + len(values)*16)
	end := dataAt
	for i := 0; i < len(values); i++ {
		if end > m.limits.Memory || len(values[i]) > m.limits.Memory-end {
			m.trap = TrapMemoryLimit
			return 0, false
		}
		end += len(values[i])
	}
	end = align8(end)
	if start < minAddress || end < start || end > m.fp {
		m.trap = TrapMemoryLimit
		return 0, false
	}
	cursor := dataAt
	for i := 0; i < len(values); i++ {
		at := start + i*16
		write32(m.memory, at, cursor)
		write32(m.memory, at+4, 0)
		write32(m.memory, at+8, len(values[i]))
		write32(m.memory, at+12, 0)
		for j := 0; j < len(values[i]); j++ {
			m.memory[cursor+j] = values[i][j]
		}
		cursor += len(values[i])
	}
	m.sp = end
	if !m.updatePeak() {
		return 0, false
	}
	return start, true
}

func (m *machine) pop() (int32, bool) {
	if m.sp-4 < m.stackBase {
		m.trap = TrapInvalidInstruction
		return 0, false
	}
	m.sp -= 4
	return int32(read32(m.memory, m.sp)), true
}

func (m *machine) load(addr int, size int) (uint32, bool) {
	accessSize := size
	if accessSize == 3 {
		accessSize = 4
	}
	if !m.address(addr, accessSize) {
		return 0, false
	}
	if size == 1 {
		return uint32(m.memory[addr]), true
	}
	if size == 2 {
		return uint32(read16(m.memory, addr)), true
	}
	return uint32(read32(m.memory, addr)), true
}

func (m *machine) store(addr int, size int, value uint32) bool {
	accessSize := size
	if accessSize == 3 {
		accessSize = 4
	}
	if !m.address(addr, accessSize) {
		return false
	}
	m.memory[addr] = byte(value)
	if size > 1 {
		m.memory[addr+1] = byte(value >> 8)
	}
	if size > 2 {
		m.memory[addr+2] = byte(value >> 16)
		m.memory[addr+3] = byte(value >> 24)
	}
	return true
}

func (m *machine) address(addr int, size int) bool {
	if addr < minAddress || size < 0 || addr > len(m.memory) || size > len(m.memory)-addr {
		m.trap = TrapMemoryLimit
		return false
	}
	return true
}

func (m *machine) buffer(addr int, size int) bool {
	if size == 0 {
		return true
	}
	return m.address(addr, size)
}

func (m *machine) charge(cost int) bool {
	if cost < 0 || m.steps > m.limits.Steps-cost {
		m.trap = TrapStepLimit
		return false
	}
	m.steps += cost
	return true
}

func (m *machine) updatePeak() bool {
	frameBytes := m.limits.Memory - m.fp
	callBytes := len(m.calls) * 20
	used := m.sp + frameBytes + callBytes
	if used > m.limits.Memory {
		m.trap = TrapMemoryLimit
		return false
	}
	if used > m.peak {
		m.peak = used
	}
	return true
}

func align8(value int) int {
	return (value + 7) &^ 7
}

func condition(flag int32, cond int) bool {
	if cond == condEq {
		return flag == 0
	}
	if cond == condNe {
		return flag != 0
	}
	if cond == condLt {
		return flag < 0
	}
	if cond == condLe {
		return flag <= 0
	}
	if cond == condGt {
		return flag > 0
	}
	return flag >= 0
}

func findRoutine(routines []routine, pc int) int {
	lo := 0
	hi := len(routines)
	for lo < hi {
		mid := (lo + hi) / 2
		if routines[mid].pc < pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(routines) && routines[lo].pc == pc {
		return lo
	}
	return -1
}

func read16(data []byte, at int) int {
	return int(data[at]) | int(data[at+1])<<8
}

func read32(data []byte, at int) int {
	return read16(data, at) | read16(data, at+2)<<16
}

func write32(data []byte, at int, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
	data[at+2] = byte(value >> 16)
	data[at+3] = byte(value >> 24)
}

func checksum(data []byte) int {
	a := 1
	b := 0
	for i := 0; i < len(data); i++ {
		a = (a + int(data[i])) % 65521
		b = (b + a) % 65521
	}
	return a | b<<16
}
