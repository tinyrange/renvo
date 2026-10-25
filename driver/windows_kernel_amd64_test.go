package driver

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"testing/fstest"

	"renvo.dev/internal/rfe/linuxuser"
	"renvo.dev/rfe/x86"
)

func compileWindowsKernelAMD64(t *testing.T, source string) []byte {
	t.Helper()
	files := fstest.MapFS{}
	for _, name := range []string{"drivers/windows_kernel_amd64.rtg", "x86_64.rtg", "elf_amd64_primitives.rtg"} {
		data, err := os.ReadFile("../backend/definitions/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files["definitions/"+name] = &fstest.MapFile{Data: data}
	}
	files["go.mod"] = &fstest.MapFile{Data: []byte("module kerneltest\n")}
	files["main.go"] = &fstest.MapFile{Data: []byte(source)}
	r, err := CompileCommand(&CommandRequest{Filesystem: memorySourceFS{files: files}, Args: []string{"-backend", "definitions/drivers/windows_kernel_amd64.rtg", "-mode=object", "-o", "driver.sys", "."}, Target: "windows-kernel/amd64", ArenaSize: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Ok {
		t.Fatalf("compile: %+v", r.Diagnostic)
	}
	return r.Binary
}

func TestCompileWindowsKernelAMD64(t *testing.T) {
	data := compileWindowsKernelAMD64(t, `package main
// renvo:linkstatic ntoskrnl.exe,KeQueryTimeIncrement
func increment() uintptr { return 0 }
//export DriverEntry
func DriverEntry(driver, path uintptr) int32 {
 if driver == 0 || path == 0 { return -1 }
 return int32(increment())
}
`)
	image, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	h, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || image.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || h.Subsystem != 1 || h.AddressOfEntryPoint == 0 || h.ImageBase != 0x140000000 {
		t.Fatalf("invalid native AMD64 driver: %+v", h)
	}
	imports, err := image.ImportedSymbols()
	if err != nil {
		t.Fatal(err)
	}
	if len(imports) != 1 || imports[0] != "KeQueryTimeIncrement:ntoskrnl.exe" {
		t.Fatalf("unexpected imports: %v", imports)
	}
	if h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXCEPTION].Size == 0 || image.Section(".pdata") == nil || h.CheckSum == 0 {
		t.Fatal("native AMD64 driver needs runtime functions and a PE checksum")
	}
}

// The entry is intentionally wider than DriverEntry: it exercises the same
// exported callback ABI used by StartDevice and SystemDisplayWrite. Both
// direct imports and function pointers must provide the Win64 caller area.
func TestWindowsKernelAMD64CallbackABI(t *testing.T) {
	data := compileWindowsKernelAMD64(t, `package main
import "unsafe"
var calls uintptr
// renvo:linkstatic ntoskrnl.exe,Fixture
func imported(a,b,c,d,e,f,g,h uintptr) uintptr { return 0 }
//export DriverEntry
func DriverEntry(a,b,c,d,e,f,g,callback uintptr) uintptr {
 var fn func(uintptr,uintptr,uintptr,uintptr,uintptr,uintptr,uintptr,uintptr) uintptr
 *(*uintptr)(unsafe.Pointer(&fn)) = callback
 calls++
 return imported(a,b,c,d,e,f,g,8)+fn(a,b,c,d,e,f,g,9)+calls
}
`)
	image, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	h := image.OptionalHeader.(*pe.OptionalHeader64)
	for _, base := range []uint64{h.ImageBase, 0x7fff00100000} {
		t.Run(fmt.Sprintf("base_%x", base), func(t *testing.T) {
			m := linuxuser.NewMemory()
			check := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			check(m.Map(base, uint64(h.SizeOfImage), 7))
			for _, section := range image.Sections {
				body, err := section.Data()
				check(err)
				check(m.WriteBytes(base+uint64(section.VirtualAddress), body))
			}
			read := func(address uint64) uint64 { value, err := m.Read(address, 8, false); check(err); return value }
			const stack = 0x200000000
			const imported = 0x7ff10000
			const callback = 0x7ff20000
			const finished = 0x7ff30000
			check(m.Map(stack, 0x10000, 3))
			imports := base + uint64(h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_IMPORT].VirtualAddress)
			iat, err := m.Read(imports+16, 4, false)
			check(err)
			check(m.Write(base+iat, 8, imported))
			c := &x86.CPU{Memory: m, Flags: 2, PC: base + uint64(h.AddressOfEntryPoint)}
			nonvolatile := []int{x86.RBX, x86.RBP, x86.RDI, x86.RSI, 12, 13, 14, 15}
			for _, reg := range nonvolatile {
				c.R[reg] = 0xabcdef0000000000 + uint64(reg)
			}
			initial := c.R
			sp := uint64(stack + 0xff88)
			c.R[x86.RSP] = sp
			check(m.Write(sp, 8, finished))
			args := []uint64{1, 2, 3, 4, 5, 6, 0x100000007, callback}
			for i, reg := range []int{x86.RCX, x86.RDX, 8, 9} {
				c.R[reg] = args[i]
			}
			for i := 4; i < len(args); i++ {
				check(m.Write(sp+40+uint64(i-4)*8, 8, args[i]))
			}
			seen := 0
			for steps := 0; c.PC != finished && steps < 10000; steps++ {
				if c.PC == imported || c.PC == callback {
					seen++
					// Unwind the suspended internal call and its exported wrapper,
					// while the evaluation stack is below the established frame.
					unwound := c.R
					var frames []string
					pc := read(unwound[x86.RSP])
					unwound[x86.RSP] += 8
					for depth := 0; pc != finished && depth < 8; depth++ {
						directory := h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXCEPTION]
						info := uint64(0)
						for at := uint64(0); at < uint64(directory.Size); at += 12 {
							entry, err := m.ReadBytes(base+uint64(directory.VirtualAddress)+at, 12)
							check(err)
							begin, end := binary.LittleEndian.Uint32(entry), binary.LittleEndian.Uint32(entry[4:])
							if base+uint64(begin) <= pc && pc < base+uint64(end) {
								info = base + uint64(binary.LittleEndian.Uint32(entry[8:]))
								break
							}
						}
						if info == 0 {
							t.Fatalf("no unwind record for %#x after %v", pc, frames)
						}
						header, err := m.ReadBytes(info, 4)
						check(err)
						if header[0] != 1 {
							t.Fatal("unexpected unwind version/flags")
						}
						codes, err := m.ReadBytes(info+4, uint64(header[2])*2)
						check(err)
						frames = append(frames, fmt.Sprintf("pc=%#x bp=%#x sp=%#x codes=%x entry=%#x", pc, unwound[x86.RBP], unwound[x86.RSP], codes, h.AddressOfEntryPoint))
						for at := 0; at < len(codes); at += 2 {
							reg, operation := codes[at+1]>>4, codes[at+1]&15
							switch operation {
							case 0:
								unwound[reg] = read(unwound[x86.RSP])
								unwound[x86.RSP] += 8
							case 3:
								unwound[x86.RSP] = unwound[header[3]&15] - uint64(header[3]>>4)*16
							default:
								t.Fatalf("unexpected unwind operation %d", operation)
							}
						}
						pc = read(unwound[x86.RSP])
						unwound[x86.RSP] += 8
					}
					if pc != finished || unwound[x86.RSP] != sp+8 {
						t.Fatalf("unwind failed pc=%#x sp=%#x", pc, unwound[x86.RSP])
					}
					for _, reg := range nonvolatile {
						if unwound[reg] != initial[reg] {
							t.Fatalf("unwind lost r%d", reg)
						}
					}
					if c.R[x86.RSP]&15 != 8 {
						t.Fatalf("foreign entry stack misaligned: %#x", c.R[x86.RSP])
					}
					for i, reg := range []int{x86.RCX, x86.RDX, 8, 9} {
						if c.R[reg] != args[i] {
							t.Fatalf("argument %d = %#x", i, c.R[reg])
						}
					}
					for i := 4; i < 7; i++ {
						if got := read(c.R[x86.RSP] + 40 + uint64(i-4)*8); got != args[i] {
							t.Fatalf("stack argument %d = %#x", i, got)
						}
					}
					last := uint64(8)
					if c.PC == callback {
						last = 9
					}
					if got := read(c.R[x86.RSP] + 64); got != last {
						t.Fatalf("eighth argument = %#x, want %d", got, last)
					}
					for i := 0; i < 4; i++ {
						check(m.Write(c.R[x86.RSP]+8+uint64(i)*8, 8, 0xdedede))
					}
					for _, reg := range []int{x86.RCX, x86.RDX, 8, 9, 10, 11} {
						c.R[reg] = 0xbad
					}
					c.PC = read(c.R[x86.RSP])
					c.R[x86.RSP] += 8
					c.R[x86.RAX] = 0x100000000 + last
					continue
				}
				if err := c.Step(); err != nil {
					code, _ := m.ReadBytes(c.PC-16, 40)
					t.Fatalf("at %#x: %v registers=%x code=%x", c.PC, err, c.R, code)
				}
			}
			if c.PC != finished || c.R[x86.RAX] != 0x200000012 || c.R[x86.RSP] != sp+8 || seen != 2 {
				t.Fatalf("bad return pc=%#x value=%#x sp=%#x calls=%d", c.PC, c.R[x86.RAX], c.R[x86.RSP], seen)
			}
			for _, reg := range nonvolatile {
				if c.R[reg] != initial[reg] {
					t.Errorf("nonvolatile r%d = %#x, want %#x", reg, c.R[reg], initial[reg])
				}
			}
			// The sorted exception directory must cover the exported entry.
			pdata, err := image.Section(".pdata").Data()
			check(err)
			found := false
			for at := 0; at < int(h.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_EXCEPTION].Size); at += 12 {
				begin, end := binary.LittleEndian.Uint32(pdata[at:]), binary.LittleEndian.Uint32(pdata[at+4:])
				if begin <= h.AddressOfEntryPoint && h.AddressOfEntryPoint < end {
					found = true
				}
				if begin >= end || at > 0 && binary.LittleEndian.Uint32(pdata[at-12:]) >= begin {
					t.Fatal("invalid sorted runtime functions")
				}
			}
			if !found {
				t.Fatal("callback is missing unwind metadata")
			}
		})
	}
}

func TestCompileWindowsKernelAMD64DisplayEntry(t *testing.T) {
	data := compileWindowsKernelAMD64(t, `package main
//export DrvEnableDriver
func DrvEnableDriver(version, size, output uintptr) uint32 { return 1 }
`)
	image, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer image.Close()
	h := image.OptionalHeader.(*pe.OptionalHeader64)
	if image.Characteristics&pe.IMAGE_FILE_DLL == 0 || h.Subsystem != 1 || h.AddressOfEntryPoint == 0 || h.MajorOperatingSystemVersion != 5 || h.MinorOperatingSystemVersion != 2 || h.MajorSubsystemVersion != 5 || h.MinorSubsystemVersion != 2 {
		t.Fatalf("invalid XP x64 display DLL: characteristics=%x header=%+v", image.Characteristics, h)
	}
	sectionEnd := binary.LittleEndian.Uint32(data[0x3c:]) + 24 + uint32(image.SizeOfOptionalHeader) + uint32(len(image.Sections))*40
	if h.SizeOfHeaders < sectionEnd+40 || image.Sections[0].Offset < h.SizeOfHeaders {
		t.Fatal("display DLL must reserve a section header for OEM resources")
	}
}
