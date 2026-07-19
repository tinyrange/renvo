package target

import "debug/elf"

const (
	elfRISCVFloatABIMask = 0x6
	elfRISCVRVE          = 0x8
)

// RV32ECILP32E returns the board-independent relocatable-object contract used
// by CH32V003-class devices and other compatible RV32E cores.
func RV32ECILP32E() ObjectTarget {
	return ObjectTarget{
		Name:        "rv32ec-ilp32e",
		Execution:   ExecutionFreestanding,
		ISA:         "rv32ec",
		ABI:         "ilp32e",
		IntBits:     32,
		PointerBits: 32,
		Format: ObjectFormat{
			Name:        "elf32-littleriscv",
			Container:   "elf",
			AddressBits: 32,
			Endian:      EndianLittle,
			MachineID:   uint16(elf.EM_RISCV),
			FlagsMask:   elfRISCVFloatABIMask | elfRISCVRVE,
			FlagsValue:  elfRISCVRVE,
		},
	}
}

// CH32V003 returns the first constrained forcing profile. It describes board
// composition only; the RV32EC instruction emitter and ilp32e object ABI remain
// reusable by other devices.
func CH32V003() Composition {
	return Composition{
		Object: RV32ECILP32E(),
		Board: Board{
			Name: "wch-ch32v003",
			Regions: []Region{
				{Name: "flash", Kind: RegionFlash, Start: 0x08000000, Size: 16 * 1024},
				{Name: "sram", Kind: RegionRAM, Start: 0x20000000, Size: 2 * 1024},
			},
			Startup: StartupContract{
				VectorSymbol:    "renvo_vectors",
				VectorAddress:   0x08000000,
				VectorAlignment: 4,
				EntryAlignment:  2,
				BSS:             BSSZeroedByStartup,
			},
			Stack: StackContract{
				InitialPointer: 0x20000800,
				DefaultSize:    512,
				GuardSize:      64,
				Alignment:      4,
				Direction:      StackGrowsDown,
			},
			Runtime: RuntimeContract{
				Operations: []string{"heap", "interrupts", "result", "volatile_memory"},
				Heap:       HeapContract{Model: HeapBump, OOM: OOMResult},
				Interrupts: InterruptContract{Model: InterruptVectored},
				Volatile: VolatileContract{
					Widths:    VolatileWidth8 | VolatileWidth16 | VolatileWidth32,
					Alignment: VolatileNaturalAligned,
				},
				Result: ResultContract{Transport: ResultTransportDebuggerMemory, Symbol: "renvores"},
			},
		},
	}
}
