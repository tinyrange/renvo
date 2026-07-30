package vm

import "testing"

func TestRunExitAndDeterministicMetrics(t *testing.T) {
	code := []byte{opMovRegImm, regRax, 7, 0, 0, 0, opExit}
	program := testProgram(code, nil, 0)
	first := Run(program, Limits{Steps: 10, Memory: 1024})
	second := Run(program, Limits{Steps: 10, Memory: 1024})
	if first.Trap != TrapNone || first.ExitCode != 7 || first.Steps != 2 || first.PeakMemory != minAddress {
		t.Fatalf("result = %+v", first)
	}
	if first.ExitCode != second.ExitCode || first.Steps != second.Steps ||
		first.PeakMemory != second.PeakMemory || first.Trap != second.Trap {
		t.Fatalf("results differ: first=%+v second=%+v", first, second)
	}
}

func TestRunEnforcesStepAndMemoryLimits(t *testing.T) {
	loop := testProgram([]byte{opJmp, 0, 0, 0, 0}, nil, 0)
	result := Run(loop, Limits{Steps: 10, Memory: 1024})
	if result.Trap != TrapStepLimit || result.Steps != 10 || result.TrapPC != 0 {
		t.Fatalf("step result = %+v", result)
	}
	static := testProgram([]byte{opExit}, nil, 1024)
	result = Run(static, Limits{Steps: 10, Memory: 1024})
	if result.Trap != TrapMemoryLimit || result.Steps != 0 {
		t.Fatalf("memory result = %+v", result)
	}
}

func TestRunRejectsCorruptArtifact(t *testing.T) {
	program := testProgram([]byte{opExit}, nil, 0)
	program[len(program)-1] ^= 1
	result := Run(program, Limits{Steps: 10, Memory: 1024})
	if result.Trap != TrapInvalidProgram {
		t.Fatalf("corrupt result = %+v", result)
	}
}

func TestLoadSizedExtensionMarkers(t *testing.T) {
	m := machine{memory: make([]byte, minAddress+3)}
	copy(m.memory[minAddress:], []byte{0x80, 0x00, 0x80})
	tests := []struct {
		size int
		addr int
		want uint32
	}{
		{size: 1, addr: minAddress, want: 0x80},
		{size: 129, addr: minAddress, want: 0xffffff80},
		{size: 2, addr: minAddress + 1, want: 0xffff8000},
		{size: 66, addr: minAddress + 1, want: 0x8000},
	}
	for _, test := range tests {
		got, ok := m.loadSized(test.addr, test.size)
		if !ok || got != test.want {
			t.Errorf("loadSized(%d, %d) = %#x, %v; want %#x, true",
				test.addr, test.size, got, ok, test.want)
		}
	}
}

func TestDirectoryDataIsSortedAndMarksDirectories(t *testing.T) {
	m := machine{files: []File{
		{Name: "/workspace/z.go"},
		{Name: "/workspace/cmd/app/main.go"},
		{Name: "/workspace/a.go"},
	}}
	data, ok := m.directoryData("/workspace")
	if !ok {
		t.Fatal("directory not found")
	}
	var names []string
	var types []byte
	for at := 0; at < len(data); {
		size := int(data[at+16]) | int(data[at+17])<<8
		end := at + 19
		for end < at+size && data[end] != 0 {
			end++
		}
		names = append(names, string(data[at+19:end]))
		types = append(types, data[at+18])
		at += size
	}
	if len(names) != 3 || names[0] != "a.go" || names[1] != "cmd" || names[2] != "z.go" {
		t.Fatalf("names = %#v", names)
	}
	if types[0] != 8 || types[1] != 4 || types[2] != 8 {
		t.Fatalf("types = %#v", types)
	}
}

func TestGetdentsReadsSynthesizedDirectory(t *testing.T) {
	m := machine{
		limits: Limits{Steps: 10000, Memory: 4096},
		memory: make([]byte, 4096),
		files: []File{
			{Name: "/workspace/z.go"},
			{Name: "/workspace/cmd/app/main.go"},
			{Name: "/workspace/a.go"},
		},
		fds: make([]descriptor, 3),
	}
	copy(m.memory[minAddress:], "/workspace\x00")
	m.regs[regRdi] = minAddress
	m.regs[regRsi] = 0
	m.regs[regRdx] = 11
	if !m.open() || m.regs[regRax] != 3 {
		t.Fatalf("open directory = %d", m.regs[regRax])
	}
	m.regs[regRax] = sysGetdents64
	m.regs[regRdi] = 3
	m.regs[regRsi] = 512
	m.regs[regRdx] = 512
	if !m.syscall() || m.regs[regRax] <= 0 {
		t.Fatalf("getdents = %d", m.regs[regRax])
	}
	size := int(m.regs[regRax])
	var names []string
	for at := 512; at < 512+size; {
		recordSize := int(m.memory[at+16]) | int(m.memory[at+17])<<8
		end := at + 19
		for end < at+recordSize && m.memory[end] != 0 {
			end++
		}
		names = append(names, string(m.memory[at+19:end]))
		at += recordSize
	}
	if len(names) != 3 || names[0] != "a.go" || names[1] != "cmd" || names[2] != "z.go" {
		t.Fatalf("names = %#v", names)
	}
	m.regs[regRax] = sysGetdents64
	if !m.syscall() || m.regs[regRax] != 0 {
		t.Fatalf("second getdents = %d", m.regs[regRax])
	}
}

func testProgram(code []byte, data []byte, bssSize int) []byte {
	program := make([]byte, headerSize+routineSize)
	copy(program, "RNVB")
	program[4] = version
	program[6] = headerSize
	write32(program, 8, len(code))
	write32(program, 12, len(data))
	write32(program, 16, minAddress)
	write32(program, 20, align8(minAddress+len(data)))
	write32(program, 24, bssSize)
	write32(program, 28, 1)
	program = append(program, code...)
	program = append(program, data...)
	write32(program, 32, checksum(program[headerSize:]))
	return program
}
