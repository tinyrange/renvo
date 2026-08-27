package uefi

import "unsafe"

const (
	AllocateAnyPages   = 0
	AllocateMaxAddress = 1
	AllocateAddress    = 2
	LoaderCode         = 1
	LoaderData         = 2
	BootServicesCode   = 3
	BootServicesData   = 4
	ConventionalMemory = 7
	UnusableMemory     = 8
	ACPIReclaimMemory  = 9
	ACPIMemoryNVS      = 10
	PersistentMemory   = 14
)

type BootServices struct{}

// AllocatePages reserves physically contiguous 4 KiB pages. For
// AllocateAddress, address contains the exact requested base. For
// AllocateMaxAddress, it contains the highest acceptable address on input.
// Firmware replaces it with the allocated base.
func AllocatePages(kind, memoryType uint32, pages uintptr, address *uint64) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil || address == nil {
		return NotReady
	}
	return Status(call4(loadWord(pointer(unsafe.Pointer(boot)), 40), uintptr(kind),
		uintptr(memoryType), pages, pointer(unsafe.Pointer(address))))
}

func FreePages(address uint64, pages uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return NotReady
	}
	return Status(call2(loadWord(pointer(unsafe.Pointer(boot)), 48), uintptr(address), pages))
}

type MemoryMap uintptr

func (m MemoryMap) Size() uintptr           { return loadWord(uintptr(m), 0) }
func (m MemoryMap) Key() uintptr            { return loadWord(uintptr(m), 8) }
func (m MemoryMap) DescriptorSize() uintptr { return loadWord(uintptr(m), 16) }
func (m MemoryMap) Version() uint32         { return load32(uintptr(m), 24) }

// GetMemoryMap writes the current firmware memory map to address. Control
// points to 32 writable bytes holding size, key, descriptor size and version.
// Passing a zero capacity queries the required size.
func GetMemoryMap(control, address, capacity uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil || control == 0 {
		return NotReady
	}
	store64(control, 0, uint64(capacity))
	store64(control, 8, 0)
	store64(control, 16, 0)
	store64(control, 24, 0)
	return Status(call5(loadWord(pointer(unsafe.Pointer(boot)), 56),
		control, address, control+8, control+16, control+24))
}

func ExitBootServices(key uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return NotReady
	}
	return Status(call2(loadWord(pointer(unsafe.Pointer(boot)), 232), uintptr(ImageHandle()), key))
}

func LocateProtocol(guid *GUID) (uintptr, Status) {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return 0, NotReady
	}
	var protocol uintptr
	status := Status(call3(loadWord(pointer(unsafe.Pointer(boot)), 320),
		pointer(unsafe.Pointer(guid)), 0, pointer(unsafe.Pointer(&protocol))))
	return protocol, status
}

func HandleProtocol(handle Handle, guid *GUID) (uintptr, Status) {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return 0, NotReady
	}
	var protocol uintptr
	status := Status(call3(loadWord(pointer(unsafe.Pointer(boot)), 152), uintptr(handle),
		pointer(unsafe.Pointer(guid)), pointer(unsafe.Pointer(&protocol))))
	return protocol, status
}

func Stall(microseconds uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return NotReady
	}
	return Status(call1(loadWord(pointer(unsafe.Pointer(boot)), 248), microseconds))
}

func AllocatePool(memoryType uint32, bytes uintptr, buffer *uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil || buffer == nil {
		return NotReady
	}
	*buffer = 0
	return Status(call3(loadWord(pointer(unsafe.Pointer(boot)), 64), uintptr(memoryType), bytes,
		pointer(unsafe.Pointer(buffer))))
}

func FreePool(buffer uintptr) Status {
	table := CurrentSystemTable()
	boot := table.Boot()
	if boot == nil {
		return NotReady
	}
	return Status(call1(loadWord(pointer(unsafe.Pointer(boot)), 72), buffer))
}

type RuntimeServices struct{}

const (
	ResetCold             = 0
	ResetWarm             = 1
	ResetShutdown         = 2
	ResetPlatformSpecific = 3
)

func Reset(kind uint32, status Status) {
	table := CurrentSystemTable()
	runtime := table.Runtime()
	if runtime == nil {
		return
	}
	call4(loadWord(pointer(unsafe.Pointer(runtime)), 104), uintptr(kind), uintptr(status), 0, 0)
}
