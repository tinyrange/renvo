# Renvo target contract

RENVO separates three machine descriptions that must not be inferred from one
another:

1. The language data model: `int`, pointer, byte, byte order, and alignment.
2. The normalized backend value model used while lowering the current IR.
3. The platform composition: object ABI, runtime operations, memory regions,
   startup, and observation transport.

The current normalized backend uses eight-byte virtual value slots. That is an
internal compatibility constraint, not a claim that every target has 64-bit
integers, pointers, or alignment. `renvoTargetProfile` records both the target
widths and the backend slot size so C, 16-bit, and non-flat-memory backends can
reject accidental coupling.

## Data-model validation

Every advertised target must have a valid profile specifying:

- `CHAR_BIT` and target byte order.
- Language `int` width.
- Data, code, and function-pointer widths.
- Maximum natural alignment.
- Flat, Harvard, segmented, or banked address model.
- Runtime and numeric capabilities.

Current Go-compatible targets use 32- or 64-bit `int`. A future 16-bit language
profile is an explicit RENVO extension and must not be advertised as ordinary Go
without documenting that incompatibility. An 8-bit device profile describes
an 8-bit byte/device class; ISO C still requires a conforming C `int` to provide
at least 16 value bits.

## Pointer address spaces

Pointer types retain an address-space identity in the backend type contract:
data, code, function, or generic. Ordinary `*T` currently produces a data
pointer. A backend for Harvard, segmented, or banked hardware must explicitly
lower conversions and must reject an unrepresentable implicit conversion.

Code and function pointers may have widths different from data pointers. Object
relocations and C carrier types must be selected from the relevant pointer
space, not from a single host `sizeof(void *)` assumption.

## Freestanding allocation and failure

A freestanding profile declares one heap model:

- no heap;
- a bounded bump/arena allocator; or
- an externally supplied allocator.

If heap support is present, out-of-memory behavior is also mandatory: trap,
return a result through the target protocol, or invoke the language panic path.
The linker/board composition owns the heap region and must prove that static
data, result storage, heap, stack reservation, and guard space fit concurrently.

## MMIO and interrupts

A target that advertises volatile memory declares the supported access widths.
Those accesses are observable operations: a C backend must emit volatile
lvalues/helpers, while a native backend must not combine, remove, widen, split,
or reorder them across other observable operations unless the target contract
explicitly permits it.

Interrupt support requires an interrupt model in addition to startup/vector
composition. The target ABI owns handler entry/return instructions, preserved
registers, stack selection/alignment, vector binding, and nesting policy.
Ordinary functions cannot be installed as handlers without this ABI adapter.

## Floating point

The existing backend represents accepted floating expressions with its legacy
scaled-integer compatibility lowering. Profiles name that behavior explicitly;
it must not be confused with IEEE-754 `float64`. A target claiming ordinary Go
floating-point or complex-number support must instead select a validated IEEE
hardware or software model and cover conversions, exceptional values,
rounding, comparisons, and ABI passing in the omnibus.

## Conformance tiers

Hosted and emulator-capable targets run the complete backend regression suite
plus format checks. Constrained targets may use staged omnibus roots, but must
still pass profile validation, object/relocation inspection, resource gates,
and the debugger-readable result contract. Passing only a smoke program is not
target support.

`backend/compiler_sources.txt` is the authoritative compiler source manifest consumed
by stage tests, benchmarks, and release builds. CI rejects any backend
implementation file missing from it.
