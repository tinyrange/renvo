package main

import (
	"fmt"
	"unsafe"

	"renvo.dev/device/uefi"
)

// LinuxBootError describes a failed x86-64 Linux boot preparation step. Status
// is the underlying firmware status when the failure came from UEFI.
type LinuxBootError struct {
	Step   string
	Status uefi.Status
}

func (e LinuxBootError) Error() string {
	if e.Status == uefi.Success {
		return e.Step
	}
	return e.Step + ": " + e.Status.Error()
}

const (
	linuxBootParamsSize = uintptr(4096)
	linuxSetupStart     = uintptr(0x1f1)
	linuxHeaderMagic    = uint32(0x53726448)
	linuxMinProtocol    = uint16(0x020c)
	xloadKernel64       = uint16(1)
	e820RAM             = uint32(1)
	e820Reserved        = uint32(2)
	e820ACPI            = uint32(3)
	e820NVS             = uint32(4)
	e820Unusable        = uint32(5)
	e820Persistent      = uint32(7)
	linuxVideoTypeEFI   = byte(0x70)
)

var (
	acpi20TableGUID = uefi.GUID{Low: 0x11d3e4f18868e871, High: 0x81883cc7800022bc}
	acpi10TableGUID = uefi.GUID{Low: 0x11d32d88eb9d2d30, High: 0x4dc13f279000169a}
)

// BootLinux64 loads a bzImage and initramfs from volume, exits UEFI boot
// services, and enters the kernel through Linux's native x86-64 boot protocol.
// It deliberately does not invoke the kernel's EFI stub.
//
// Success does not return. The volume is consumed and closed after both files
// have been read; the caller should close every other firmware resource first.
func BootLinux64(volume *uefi.File, kernelPath, initramfsPath, commandLine string) *LinuxBootError {
	fmt.Println("Loading kernel " + kernelPath + "...")
	kernel, status := volume.Open(kernelPath, uefi.FileModeRead, 0)
	if status.Failed() {
		return &LinuxBootError{"open kernel " + kernelPath, status}
	}
	header := make([]byte, 4096)
	headerBytes, status := kernel.Read(header)
	if status.Failed() {
		kernel.Close()
		return &LinuxBootError{"read kernel header", status}
	}
	if headerBytes < 0x268 || uint32At(header, 0x202) != linuxHeaderMagic {
		kernel.Close()
		return &LinuxBootError{"kernel is not a Linux bzImage", uefi.Success}
	}
	protocol := uint16At(header, 0x206)
	if protocol < linuxMinProtocol {
		kernel.Close()
		return &LinuxBootError{"kernel boot protocol is older than 2.12", uefi.Success}
	}
	if header[0x211]&1 == 0 {
		kernel.Close()
		return &LinuxBootError{"kernel is not a bzImage", uefi.Success}
	}
	if uint16At(header, 0x236)&xloadKernel64 == 0 {
		kernel.Close()
		return &LinuxBootError{"kernel has no x86-64 entry point", uefi.Success}
	}
	if header[0x234] == 0 {
		kernel.Close()
		return &LinuxBootError{"kernel is not relocatable", uefi.Success}
	}

	setupSectors := uintptr(header[0x1f1])
	if setupSectors == 0 {
		setupSectors = 4
	}
	kernelOffset := uint64((setupSectors + 1) * 512)
	loadSize := uint64(uint32At(header, 0x260))
	if loadSize == 0 {
		kernel.Close()
		return &LinuxBootError{"kernel does not report an initialization size", uefi.Success}
	}
	alignment := uint64(uint32At(header, 0x230))
	kernelAddress, status := allocateAlignedKernel(uintptr(loadSize), uintptr(alignment))
	if status.Failed() {
		kernel.Close()
		return &LinuxBootError{"allocate aligned kernel memory", status}
	}
	zeroMemory(uintptr(kernelAddress), uintptr(loadSize))
	status = kernel.SetPosition(kernelOffset)
	if status.Failed() {
		kernel.Close()
		return &LinuxBootError{"seek kernel payload", status}
	}
	protectedSize, status := readFileAddressUntilEOF(kernel, uintptr(kernelAddress), uintptr(loadSize))
	kernel.Close()
	if status.Failed() || protectedSize == 0 {
		return &LinuxBootError{"load kernel payload", status}
	}

	fmt.Println("Loading initramfs " + initramfsPath + "...")
	initrd, status := volume.Open(initramfsPath, uefi.FileModeRead, 0)
	if status.Failed() {
		return &LinuxBootError{"open initramfs " + initramfsPath, status}
	}
	const initrdCapacity = uintptr(64 * 1024 * 1024)
	initrdMaximum := uint64(uint32At(header, 0x22c))
	if initrdMaximum == 0 {
		initrdMaximum = 0x7fffffff
	}
	initrdAddress := initrdMaximum
	status = uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderData,
		pagesFor(initrdCapacity), &initrdAddress)
	if status.Failed() {
		initrd.Close()
		return &LinuxBootError{"allocate initramfs", status}
	}
	initrdSize, status := readFileAddressUntilEOF(initrd, uintptr(initrdAddress), initrdCapacity)
	initrd.Close()
	if status.Failed() || initrdSize == 0 {
		return &LinuxBootError{"load initramfs", status}
	}
	volume.Close()

	paramsAddress := uint64(0xffffffff)
	status = uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderData,
		pagesFor(linuxBootParamsSize), &paramsAddress)
	if status.Failed() {
		return &LinuxBootError{"allocate Linux boot parameters", status}
	}
	params := uintptr(paramsAddress)
	zeroMemory(params, linuxBootParamsSize)
	framebuffer := uefi.Framebuffer{}
	graphics, graphicsStatus := uefi.GraphicsOutput()
	if !graphicsStatus.Failed() {
		framebuffer, graphicsStatus = graphics.Framebuffer()
		if !graphicsStatus.Failed() {
			populateLinuxScreenInfo(params, framebuffer)
		}
	}
	acpi := uefi.CurrentSystemTable().ConfigurationTable(acpi20TableGUID)
	if acpi == 0 {
		acpi = uefi.CurrentSystemTable().ConfigurationTable(acpi10TableGUID)
	}
	if acpi != 0 {
		store64(params, 0x70, uint64(acpi))
	}
	headerEnd := uintptr(0x202 + int(header[0x201]))
	if headerEnd > uintptr(len(header)) || headerEnd < 0x268 {
		return &LinuxBootError{"kernel setup header is truncated", uefi.Success}
	}
	for at := linuxSetupStart; at < headerEnd; at++ {
		store8(params, at, header[at])
	}
	store8(params, 0x210, 0xff) // unassigned boot-loader identifier
	store32(params, 0x214, uint32(kernelAddress))
	store32(params, 0x218, uint32(initrdAddress))
	store32(params, 0x21c, uint32(initrdSize))
	store32(params, 0x0c0, uint32(initrdAddress>>32))
	store32(params, 0x0c4, 0)

	commandBytes := []byte(commandLine)
	maximumCommand := uintptr(uint32At(header, 0x238))
	if maximumCommand == 0 {
		maximumCommand = 255
	}
	if uintptr(len(commandBytes)) > maximumCommand {
		return &LinuxBootError{"kernel command line is too long", uefi.Success}
	}
	commandAddress := uint64(0xffffffff)
	status = uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderData,
		pagesFor(uintptr(len(commandBytes)+1)), &commandAddress)
	if status.Failed() {
		return &LinuxBootError{"allocate kernel command line", status}
	}
	for i := 0; i < len(commandBytes); i++ {
		store8(uintptr(commandAddress), uintptr(i), commandBytes[i])
	}
	store8(uintptr(commandAddress), uintptr(len(commandBytes)), 0)
	store32(params, 0x228, uint32(commandAddress))
	store32(params, 0x0c8, uint32(commandAddress>>32))

	stackAddress := uint64(0xffffffff)
	status = uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderData, 16, &stackAddress)
	if status.Failed() {
		return &LinuxBootError{"allocate kernel entry stack", status}
	}

	// The native 64-bit boot protocol requires the kernel initialization
	// range, boot parameters, command line, and entry stack to be identity
	// mapped. Do not inherit this state from UEFI: real firmware is free to use
	// mappings unlike OVMF's. Six pages describe the first 4 GiB with 2 MiB
	// leaves. A seventh provides the optional five-level root used when UEFI
	// left CR4.LA57 set, and an eighth holds the transition and trampoline.
	pageTables := uint64(0xffffffff)
	status = uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderCode, 8, &pageTables)
	if status.Failed() {
		return &LinuxBootError{"allocate Linux identity page tables", status}
	}
	prepareIdentityMap(uintptr(pageTables))
	transition := uintptr(pageTables + 7*4096)
	trampoline := prepareLinuxTransition(transition)
	store64(transition, 0, kernelAddress+0x200)
	store64(transition, 8, uint64(params))
	store64(transition, 16, stackAddress+16*4096)
	store64(transition, 24, pageTables)
	store64(transition, 32, pageTables+6*4096)
	store64(transition, 40, uint64(trampoline))

	fmt.Println("Starting Linux...")

	const mapCapacity = uintptr(65536)
	var mapControl [4]uintptr
	mapBytes := make([]byte, mapCapacity)
	controlAddress := uintptr(unsafe.Pointer(&mapControl[0]))
	mapAddress := uintptr(unsafe.Pointer(&mapBytes[0]))
	memoryMap := uefi.MemoryMap(controlAddress)

	for attempt := 0; attempt < 3; attempt++ {
		status = uefi.GetMemoryMap(controlAddress, mapAddress, mapCapacity)
		if status.Failed() {
			return &LinuxBootError{"read UEFI memory map", status}
		}
		populateLinuxMemoryInfo(params, mapAddress, memoryMap.Size(),
			memoryMap.DescriptorSize(), memoryMap.Version())
		status = uefi.ExitBootServices(memoryMap.Key())
		if !status.Failed() {
			markFirmwareExit(framebuffer)
			enterLinux64(transition)
			return &LinuxBootError{"Linux kernel returned", uefi.LoadError}
		}
	}
	return &LinuxBootError{"exit UEFI boot services", status}
}

// prepareIdentityMap builds four- and five-level roots covering physical
// addresses [0, 4 GiB) with writable 2 MiB pages. Every address handed to the
// Linux entry point is deliberately allocated below 4 GiB.
func prepareIdentityMap(root uintptr) {
	zeroMemory(root, 7*4096)
	store64(root, 0, uint64(root+4096)|3)
	for gigabyte := uintptr(0); gigabyte < 4; gigabyte++ {
		directory := root + (2+gigabyte)*4096
		store64(root+4096, gigabyte*8, uint64(directory)|3)
		for page := uintptr(0); page < 512; page++ {
			physical := (gigabyte << 30) + (page << 21)
			store64(directory, page*8, uint64(physical)|0x83)
		}
	}
	store64(root+6*4096, 0, uint64(root)|3)
}

// prepareLinuxTransition installs the few instructions which must execute
// immediately after CR3 changes. Keeping this trampoline below 4 GiB means its
// next instruction is covered by both the firmware and Linux page tables.
func prepareLinuxTransition(page uintptr) uintptr {
	trampoline := page + 64
	code := [...]byte{
		0x0f, 0x20, 0xe0, // MOV RAX, CR4
		0xa9, 0x00, 0x10, 0x00, 0x00, // TEST EAX, CR4.LA57
		0x74, 0x03, // JZ use four-level root already in RDI
		0x4c, 0x89, 0xd7, // MOV RDI, R10 (five-level root)
		0x0f, 0x22, 0xdf, // MOV CR3, RDI
		0x31, 0xed, // XOR EBP, EBP
		0x31, 0xdb, // XOR EBX, EBX
		0x31, 0xff, // XOR EDI, EDI
		0x41, 0xff, 0xe0, // JMP R8
	}
	for i := 0; i < len(code); i++ {
		store8(trampoline, uintptr(i), code[i])
	}
	return trampoline
}

func populateLinuxScreenInfo(params uintptr, framebuffer uefi.Framebuffer) {
	store8(params, 0x0f, linuxVideoTypeEFI)
	store16(params, 0x12, uint16(framebuffer.Width))
	store16(params, 0x14, uint16(framebuffer.Height))
	store16(params, 0x16, 32)
	store32(params, 0x18, uint32(framebuffer.Address))
	store32(params, 0x1c, framebuffer.Stride*framebuffer.Height*4)
	store16(params, 0x24, uint16(framebuffer.Stride*4))
	store8(params, 0x26, 8)
	store8(params, 0x28, 8)
	store8(params, 0x29, 8)
	store8(params, 0x2a, 8)
	store8(params, 0x2c, 8)
	store8(params, 0x2d, 24)
	if framebuffer.PixelFormat == uefi.PixelRedGreenBlueReserved8BitPerColor {
		store8(params, 0x27, 0)
		store8(params, 0x2b, 16)
	} else {
		store8(params, 0x27, 16)
		store8(params, 0x2b, 0)
	}
	store16(params, 0x32, 1)
	if uint64(framebuffer.Address)>>32 != 0 {
		store32(params, 0x36, 2)
		store32(params, 0x3a, uint32(uint64(framebuffer.Address)>>32))
	}
}

// markFirmwareExit draws a cherry-red strip after ExitBootServices has
// succeeded. It uses only the framebuffer address captured earlier: no
// firmware service is legal at this point.
func markFirmwareExit(framebuffer uefi.Framebuffer) {
	if framebuffer.Address == 0 {
		return
	}
	height := 8
	if int(framebuffer.Height) < height {
		height = int(framebuffer.Height)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < int(framebuffer.Width); x++ {
			framebuffer.Set(x, y, 182, 36, 79)
		}
	}
}

func pagesFor(bytes uintptr) uintptr { return (bytes + 4095) >> 12 }

func allocateAlignedKernel(size, alignment uintptr) (uint64, uefi.Status) {
	if alignment < 4096 {
		alignment = 4096
	}
	if alignment&(alignment-1) != 0 {
		return 0, uefi.InvalidParameter
	}
	wantedPages := pagesFor(size)
	extraPages := alignment >> 12
	base := uint64(0xffffffff)
	status := uefi.AllocatePages(uefi.AllocateMaxAddress, uefi.LoaderData,
		wantedPages+extraPages, &base)
	if status.Failed() {
		return 0, status
	}
	aligned := (uintptr(base) + alignment - 1) & ^(alignment - 1)
	prefixPages := (aligned - uintptr(base)) >> 12
	if prefixPages != 0 {
		uefi.FreePages(base, prefixPages)
	}
	suffixPages := wantedPages + extraPages - prefixPages - wantedPages
	if suffixPages != 0 {
		uefi.FreePages(uint64(aligned+(wantedPages<<12)), suffixPages)
	}
	return uint64(aligned), uefi.Success
}

func readFileAddressUntilEOF(file *uefi.File, address, capacity uintptr) (uintptr, uefi.Status) {
	total := uintptr(0)
	for total < capacity {
		chunk := capacity - total
		if chunk > 1024*1024 {
			chunk = 1024 * 1024
		}
		read, status := file.ReadAddress(address, chunk)
		if status.Failed() {
			return total, status
		}
		if read == 0 {
			return total, uefi.Success
		}
		address += read
		total += read
	}
	return total, uefi.BadBufferSize
}

func populateLinuxMemoryInfo(params, memoryMap, mapSize, descriptorSize uintptr, _ uint32) {
	// This loader deliberately leaves efi_info empty. Advertising EFI there
	// also promises Linux a preserved UEFI memory map and usable runtime-service
	// mappings. The native boot path supplies E820, ACPI, and screen_info
	// directly instead.
	entries := 0
	for at := uintptr(0); at+descriptorSize <= mapSize && entries < 128; at += descriptorSize {
		descriptor := memoryMap + at
		address := load64(descriptor, 8)
		size := load64(descriptor, 24) * 4096
		kind := linuxE820Type(load32(descriptor, 0))
		if size == 0 {
			continue
		}
		if entries != 0 {
			previous := params + 0x2d0 + uintptr(entries-1)*20
			if load32(previous, 16) == kind && load64(previous, 0)+load64(previous, 8) == address {
				store64(previous, 8, load64(previous, 8)+size)
				continue
			}
		}
		entry := params + 0x2d0 + uintptr(entries)*20
		store64(entry, 0, address)
		store64(entry, 8, size)
		store32(entry, 16, kind)
		entries++
	}
	store8(params, 0x1e8, byte(entries))
}

func linuxE820Type(memoryType uint32) uint32 {
	switch memoryType {
	case uefi.LoaderCode, uefi.LoaderData, uefi.BootServicesCode, uefi.BootServicesData, uefi.ConventionalMemory:
		return e820RAM
	case uefi.ACPIReclaimMemory:
		return e820ACPI
	case uefi.ACPIMemoryNVS:
		return e820NVS
	case uefi.UnusableMemory:
		return e820Unusable
	case uefi.PersistentMemory:
		return e820Persistent
	}
	return e820Reserved
}

func uint16At(data []byte, at int) uint16 {
	return uint16(data[at]) | uint16(data[at+1])<<8
}

func uint32At(data []byte, at int) uint32 {
	return uint32(data[at]) | uint32(data[at+1])<<8 |
		uint32(data[at+2])<<16 | uint32(data[at+3])<<24
}

func load32(base, offset uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(base + offset))
}

func load64(base, offset uintptr) uint64 {
	return *(*uint64)(unsafe.Pointer(base + offset))
}

func store8(base, offset uintptr, value byte) {
	*(*byte)(unsafe.Pointer(base + offset)) = value
}

func store16(base, offset uintptr, value uint16) {
	*(*uint16)(unsafe.Pointer(base + offset)) = value
}

func store32(base, offset uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(base + offset)) = value
}

func store64(base, offset uintptr, value uint64) {
	*(*uint64)(unsafe.Pointer(base + offset)) = value
}
