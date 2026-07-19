# Freestanding target composition

The `target` package represents firmware construction as two explicit layers.
An `ObjectTarget` fixes execution model, ISA, ABI, language widths, byte order,
object class/machine, and ABI-significant header flags without selecting a
board. A `Board` separately fixes memory regions, startup and BSS behavior,
entry/vector alignment, stack direction and guard, heap/OOM behavior,
interrupt and volatile-memory contracts, debugger result transport, and the
small set of runtime imports it intentionally supplies. Vector and debugger
result symbol names are part of the contract rather than caller-selected
defaults. `Composition` combines
the two only when constructing a final firmware image.

`RV32ECILP32E` is therefore usable by an object emitter on its own. Both the C
reference path and a native emitter consume that object contract. The C89
profile is derived from it, and a bring-up plan rejects a toolchain whose
declared object format or ABI disagrees with the composition.

`target.Validate` runs before simulation or flashing. It rejects sections or
load images outside their regions, VMA overlap, reserved-memory overlap,
stack/guard collisions, bad entry and vector placement or alignment, implicit
hosted imports, object-machine/ABI mismatch, and aggregate flash/RAM budget
overflow. Reserved ranges reduce usable capacity instead of appearing as free
headroom. Its `Usage` result reports static RAM, heap, stack, guard, flash use,
and remaining headroom separately.

`target.ArtifactFromELF` derives that validation input from a linked ELF image.
It records allocatable section VMA and flags, obtains LMA from `PT_LOAD`
physical addresses, keeps `SHT_NOBITS` out of the flash image, locates the
descriptor-selected vector symbol, and carries unresolved imports into the
forbidden-runtime check. ELF class, byte order, machine, and ABI-significant
`e_flags` come from the linked image rather than caller claims. Relocatable
objects are rejected because board budgets must be evaluated against the actual
linked image.

`target.CH32V003` combines the reusable RV32EC/ILP32E object target with 16 KiB
flash and 2 KiB SRAM. Its contract requires the ELF32 RISC-V E/soft-float ABI,
two-byte entry alignment, startup BSS zeroing, a downward four-byte-aligned
stack, natural 8/16/32-bit volatile accesses, and the `renvores` debugger-memory
result block. It intentionally supplies no hosted imports. Startup/vector code
and debugger result transport belong to the board layer, never to an RV32
emitter.

This establishes the host-side composition and budget gate from #23. Linking
the C-reference and native omnibus objects remains dependent on #19, #20, and
#22, so #23 stays open until those artifacts exercise this descriptor.
