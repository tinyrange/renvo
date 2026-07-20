package main

import (
	"bytes"
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var executableLayoutSmokeSource = []byte(`package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)

func TestLinuxImagesSeparateExecutableAndWritableLoads(t *testing.T) {
	for _, target := range []string{"linux/amd64", "linux/386", "linux/aarch64", "linux/arm"} {
		image, ok := RenvoCompileSourceToBytesStrip(executableLayoutSmokeSource, target, false)
		if !ok {
			t.Fatalf("compile %s", target)
		}
		file, err := elf.NewFile(bytes.NewReader(image))
		if err != nil {
			t.Fatalf("parse %s: %v", target, err)
		}
		wantType := elf.ET_EXEC
		if target == "linux/amd64" || target == "linux/aarch64" || target == "linux/arm" {
			wantType = elf.ET_DYN
		}
		if file.Type != wantType {
			t.Errorf("%s ELF type = %s, want %s", target, file.Type, wantType)
		}
		rxLoads := 0
		rwLoads := 0
		for _, program := range file.Progs {
			if program.Type != elf.PT_LOAD {
				continue
			}
			if program.Flags&elf.PF_X != 0 && program.Flags&elf.PF_W != 0 {
				t.Errorf("%s has writable+executable PT_LOAD: %#v", target, program.ProgHeader)
			}
			if program.Align > 1 && program.Off%program.Align != program.Vaddr%program.Align {
				t.Errorf("%s incongruent PT_LOAD offset/address: %#v", target, program.ProgHeader)
			}
			if program.Filesz > program.Memsz {
				t.Errorf("%s PT_LOAD file size exceeds memory size: %#v", target, program.ProgHeader)
			}
			if program.Flags == (elf.PF_R | elf.PF_X) {
				rxLoads++
			}
			if program.Flags == (elf.PF_R | elf.PF_W) {
				rwLoads++
			}
		}
		if rxLoads != 1 || rwLoads != 1 {
			t.Errorf("%s load policy = %d RX/%d RW, want one of each", target, rxLoads, rwLoads)
		}
		rodata := file.Section(".rodata")
		if rodata == nil || rodata.Flags&elf.SHF_ALLOC == 0 || rodata.Flags&elf.SHF_WRITE != 0 {
			t.Errorf("%s .rodata flags = %#v, want allocated read-only data", target, rodata)
		}
		bss := file.Section(".bss")
		if bss == nil || bss.Flags&elf.SHF_WRITE == 0 || bss.Flags&elf.SHF_EXECINSTR != 0 {
			t.Errorf("%s .bss flags = %#v, want writable non-executable storage", target, bss)
		}
		file.Close()
	}
}

func TestLinuxNativePIELoadsAtRandomizedAddresses(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("requires a Linux amd64 or arm64 host")
	}
	policy, err := os.ReadFile("/proc/sys/kernel/randomize_va_space")
	if err != nil || strings.TrimSpace(string(policy)) == "0" {
		t.Skip("kernel address randomization is disabled")
	}
	source := []byte("package main\nfunc appMain() int { for {} }\n")
	target := "linux/amd64"
	if runtime.GOARCH == "arm64" {
		target = "linux/aarch64"
	}
	image, ok := RenvoCompileSourceToBytesStrip(source, target, true)
	if !ok {
		t.Fatalf("compile %s PIE", target)
	}
	path := t.TempDir() + "/pie-loop"
	if err := os.WriteFile(path, image, 0o755); err != nil {
		t.Fatal(err)
	}
	bases := make(map[string]bool)
	for attempt := 0; attempt < 4; attempt++ {
		cmd := exec.Command(path)
		if err := cmd.Start(); err != nil {
			t.Fatalf("start PIE: %v", err)
		}
		base := ""
		var readErr error
		for poll := 0; poll < 20 && base == ""; poll++ {
			var maps []byte
			maps, readErr = os.ReadFile("/proc/" + strconv.Itoa(cmd.Process.Pid) + "/maps")
			if readErr != nil {
				break
			}
			for _, line := range strings.Split(string(maps), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 6 && fields[1] == "r-xp" && fields[len(fields)-1] == path {
					base = strings.Split(fields[0], "-")[0]
					break
				}
			}
			if base == "" {
				time.Sleep(time.Millisecond)
			}
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if readErr != nil {
			t.Fatalf("read process mappings: %v", readErr)
		}
		if base == "" {
			t.Fatal("PIE executable mapping was not found")
		}
		bases[base] = true
	}
	if len(bases) < 2 {
		t.Fatalf("PIE used one load address across four runs: %v", bases)
	}
}

func TestLinuxArmPIERunsAtAlternateBases(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires Linux qemu-user")
	}
	qemu, err := exec.LookPath("qemu-arm")
	if err != nil {
		t.Skip("qemu-arm is not installed")
	}
	image, ok := RenvoCompileSourceToBytesStrip(executableLayoutSmokeSource, "linux/arm", true)
	if !ok {
		t.Fatal("compile Linux ARM PIE")
	}
	path := t.TempDir() + "/arm-pie"
	if err := os.WriteFile(path, image, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"0x10000000", "0x20000000", "0x30000000"} {
		output, err := exec.Command(qemu, "-B", base, path).CombinedOutput()
		if err != nil {
			t.Fatalf("run ARM PIE at guest base %s: %v: %s", base, err, output)
		}
		if string(output) != "PASS\n" {
			t.Fatalf("ARM PIE at guest base %s output = %q", base, output)
		}
	}
}

func TestWindowsImagesKeepCodeAndDataNonOverlapping(t *testing.T) {
	for _, target := range []string{"windows/amd64", "windows/386", "windows/arm64"} {
		image, ok := RenvoCompileSourceToBytesStrip(executableLayoutSmokeSource, target, true)
		if !ok {
			t.Fatalf("compile %s", target)
		}
		file, err := pe.NewFile(bytes.NewReader(image))
		if err != nil {
			t.Fatalf("parse %s: %v", target, err)
		}
		for _, section := range file.Sections {
			executable := section.Characteristics&pe.IMAGE_SCN_MEM_EXECUTE != 0
			writable := section.Characteristics&pe.IMAGE_SCN_MEM_WRITE != 0
			if executable && writable {
				t.Errorf("%s section %s is writable+executable", target, section.Name)
			}
		}
		dynamicBase := uint16(0)
		nxCompatible := uint16(0)
		baseRelocationSize := uint32(0)
		switch header := file.OptionalHeader.(type) {
		case *pe.OptionalHeader32:
			dynamicBase = header.DllCharacteristics & 0x40
			nxCompatible = header.DllCharacteristics & 0x100
			baseRelocationSize = header.DataDirectory[5].Size
		case *pe.OptionalHeader64:
			dynamicBase = header.DllCharacteristics & 0x40
			nxCompatible = header.DllCharacteristics & 0x100
			baseRelocationSize = header.DataDirectory[5].Size
		default:
			t.Fatalf("%s optional header = %T", target, file.OptionalHeader)
		}
		if dynamicBase != 0 {
			t.Errorf("%s advertises ASLR without a base relocation table", target)
		}
		if baseRelocationSize != 0 {
			t.Errorf("%s has an unexpected base relocation directory", target)
		}
		if nxCompatible == 0 {
			t.Errorf("%s does not advertise NX compatibility", target)
		}
		file.Close()
	}
}

func TestDarwinImageUsesPIEWithWXSegmentPolicy(t *testing.T) {
	image, ok := RenvoCompileSourceToBytesStrip(executableLayoutSmokeSource, "darwin/arm64", true)
	if !ok {
		t.Fatal("compile darwin/arm64")
	}
	file, err := macho.NewFile(bytes.NewReader(image))
	if err != nil {
		t.Fatalf("parse Darwin image: %v", err)
	}
	if file.Flags&macho.FlagPIE == 0 {
		t.Error("Darwin image does not advertise PIE")
	}
	for _, load := range file.Loads {
		segment, ok := load.(*macho.Segment)
		if !ok {
			continue
		}
		executable := segment.Prot&4 != 0
		writable := segment.Prot&2 != 0
		if executable && writable {
			t.Errorf("Darwin segment %s is writable+executable", segment.Name)
		}
	}
	file.Close()
}
