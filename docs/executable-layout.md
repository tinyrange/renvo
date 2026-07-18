# Hosted executable layout

RTG treats executable layout as a target policy, separate from the language
ABI. Hosted images never require writable executable memory.

## Linux ELF

Linux outputs are fixed-address `ET_EXEC` images. Their first `PT_LOAD` maps
the ELF headers, code, and immutable `.rodata` with read/execute permission.
The second maps page-aligned zero-fill `.bss` with read/write permission. The
zero-fill load has no file payload, so separating permissions costs one program
header rather than a page of padding in every small binary.

PIE is deliberately disabled for now. The amd64 and AArch64 emitters already
use mostly PC-relative addressing, but 386 and ARM still materialize absolute
addresses. Advertising PIE before every architecture has a relocation or
position-independent addressing contract would create images that only appear
to support ASLR. Linux keeps its established load addresses until that complete
contract is implemented.

## Windows PE

PE outputs use an RX `.text` section and an RW `.data`/BSS section and advertise
NX compatibility. They remain fixed-base images and do not set `DYNAMIC_BASE`,
because RTG does not emit a base-relocation directory yet. This is explicit for
x86, x86-64, and ARM64 rather than relying on loader-specific fallback.

## Darwin Mach-O

Darwin/ARM64 outputs use separate `__TEXT` (RX), `__DATA` (RW), and
`__LINKEDIT` (R) segments. They advertise `MH_PIE`; code/data references and
import stubs use ARM64 PC-relative page addressing and therefore tolerate the
dyld slide.

## WASI and freestanding targets

WASI stores mutable state in WebAssembly linear memory while executable code is
owned by the embedding engine, so native W^X segment flags do not apply.

Freestanding image layouts remain a board/target-profile decision: flash may be
RX or R, RAM is RW, and a profile can use fixed physical addresses. That policy
does not weaken the hosted defaults above, and board artifact gates continue to
validate their declared memory regions independently.
