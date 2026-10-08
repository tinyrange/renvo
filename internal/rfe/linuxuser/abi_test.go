package linuxuser

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// A non-A64 ELF with a halfword-aligned entry exercises the adapter boundary;
// no instruction decoder or host syscall is involved in this loader test.
func tinyELF() []byte {
	image := make([]byte, 0x106)
	copy(image, "\x7fELF")
	image[4], image[5], image[6] = 2, 1, 1
	binary.LittleEndian.PutUint16(image[16:], 2)
	binary.LittleEndian.PutUint16(image[18:], 243)
	binary.LittleEndian.PutUint32(image[20:], 1)
	binary.LittleEndian.PutUint64(image[24:], 0x1102)
	binary.LittleEndian.PutUint64(image[32:], 64)
	binary.LittleEndian.PutUint32(image[48:], 1)
	binary.LittleEndian.PutUint16(image[52:], 64)
	binary.LittleEndian.PutUint16(image[54:], 56)
	binary.LittleEndian.PutUint16(image[56:], 1)
	binary.LittleEndian.PutUint32(image[64:], 1)
	binary.LittleEndian.PutUint32(image[68:], 5)
	binary.LittleEndian.PutUint64(image[80:], 0x1000)
	binary.LittleEndian.PutUint64(image[96:], uint64(len(image)))
	binary.LittleEndian.PutUint64(image[104:], 4096)
	binary.LittleEndian.PutUint64(image[112:], 4096)
	return image
}
func TestELFGuestABIAndEntropy(t *testing.T) {
	abi := ELFABI{Machine: 243, AllowedFlags: 1, EntryAlignment: 2, EntryBytes: 2}
	fill := func(b []byte) error {
		for i := range b {
			b[i] = byte(i + 1)
		}
		return nil
	}
	p, err := Load(tinyELF(), []string{"tiny"}, abi, fill)
	if err != nil || p.Entry != 0x1102 || p.StackPointer&15 != 0 {
		t.Fatal(p, err)
	}
	found := false
	// argc, argv[0], NULL, envp NULL, then key/value auxiliary records.
	for at := p.StackPointer + 32; at < StackTop; at += 16 {
		key, err := p.Memory.Read(at, 8, false)
		if err != nil {
			t.Fatal(err)
		}
		if key == 0 {
			break
		}
		if key == 25 {
			address, err := p.Memory.Read(at+8, 8, false)
			if err != nil {
				t.Fatal(err)
			}
			data, err := p.Memory.ReadBytes(address, 16)
			if err != nil {
				t.Fatal(err)
			}
			for i, v := range data {
				if v != byte(i+1) {
					t.Fatal("entropy changed", data)
				}
			}
			found = true
		}
	}
	if !found {
		t.Fatal("AT_RANDOM missing")
	}
	wrongMachine, wrongFlags, wrongAlignment := abi, abi, abi
	wrongMachine.Machine = 183
	wrongFlags.AllowedFlags = 0
	wrongAlignment.EntryAlignment = 4
	for _, bad := range []ELFABI{wrongMachine, wrongFlags, wrongAlignment} {
		if p, err = Load(tinyELF(), nil, bad, fill); err == nil || p != nil {
			t.Fatal("accepted incompatible ELF ABI", bad)
		}
	}
	if p, err = Load(tinyELF(), nil, abi, nil); err == nil || p != nil {
		t.Fatal("accepted missing entropy")
	}
	if p, err = Load(tinyELF(), nil, abi, func([]byte) error { return fmt.Errorf("entropy unavailable") }); err == nil || p != nil {
		t.Fatal("ignored entropy failure")
	}
}
func TestSyscallRegisterAdapterAndClockCapability(t *testing.T) {
	m := NewMemory()
	if err := m.Map(4096, 4096, 3); err != nil {
		t.Fatal(err)
	}
	p := Process{Memory: m, Clock: func(id uint64) (int64, int64, bool) { return 123 + int64(id), 456, true }}
	abi := SyscallABI{Number: 7, Result: 6, Arguments: [6]int{5, 4, 3, 2, 1, 0}, Trap: func(err error) (bool, error) { return false, err }}
	state := []uint64{10, 11, 12, 13, 4096, 1, 77, 113}
	original := append([]uint64(nil), state...)
	exited, _, err := p.ApplySyscall(state, abi, nil, nil)
	if err != nil || exited || state[6] != 0 {
		t.Fatal(state, err)
	}
	for i := range state {
		if i != 6 && state[i] != original[i] {
			t.Fatal("clobbered argument", i)
		}
	}
	seconds, _ := m.Read(4096, 8, false)
	nanos, _ := m.Read(4104, 8, false)
	if seconds != 124 || nanos != 456 {
		t.Fatal("clock encoding", seconds, nanos)
	}
	p.Clock = nil
	p.ApplySyscall(state, abi, nil, nil)
	if state[6] != errno(38) {
		t.Fatal("missing clock capability")
	}
	p.Clock = func(uint64) (int64, int64, bool) { return 0, 1000000000, true }
	p.ApplySyscall(state, abi, nil, nil)
	seconds, _ = m.Read(4096, 8, false)
	if state[6] != errno(5) || seconds != 124 {
		t.Fatal("invalid clock partially committed")
	}
	state[7], state[5] = 93, 19
	exited, code, err := p.ApplySyscall(state, abi, nil, nil)
	if !exited || code != 19 || err != nil {
		t.Fatal("exit adapter", exited, code, err)
	}
	abi.Result = 99
	before := append([]uint64(nil), state...)
	if _, _, err = p.ApplySyscall(state, abi, nil, nil); err == nil {
		t.Fatal("invalid register accepted")
	}
	for i := range state {
		if state[i] != before[i] {
			t.Fatal("invalid ABI mutated state")
		}
	}
}
