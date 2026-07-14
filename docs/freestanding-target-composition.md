# Freestanding target composition

The `target` package separates a reusable ISA/object backend from board image
composition. Native and C-reference objects describe sections, load addresses,
entry/vector placement, unresolved imports, heap reservation, and stack needs.
The board descriptor owns memory regions, startup placement, the object ABI,
and the small set of runtime imports it intentionally supplies.

`target.Validate` runs before simulation or flashing. It rejects sections or
load images outside their regions, VMA overlap, reserved-memory overlap,
stack/guard collisions, bad entry and vector placement, implicit hosted
imports, and aggregate flash/RAM budget overflow. Its `Usage` result reports
static RAM, heap, stack, guard, flash use, and remaining headroom separately.

`target.ArtifactFromELF` derives that validation input from a linked ELF image.
It records allocatable section VMA and flags, obtains LMA from `PT_LOAD`
physical addresses, keeps `SHT_NOBITS` out of the flash image, locates the
descriptor-selected vector symbol, and carries unresolved imports into the
forbidden-runtime check. It rejects relocatable objects because board budgets
must be evaluated against the actual linked image.

`target.CH32V003` is the initial forcing profile: reusable RV32EC instruction
selection, the ilp32e ABI, ELF32 little-endian objects, 16 KiB flash, and 2 KiB
SRAM. It intentionally supplies no hosted imports. Startup/vector code and
debugger result transport belong to the board layer, never to an RV32 emitter.

This establishes the host-side composition and budget gate from #23. Linking
the C-reference and native omnibus objects remains dependent on #19, #20, and
#22, so #23 stays open until those artifacts exercise this descriptor.
