# RTG backend contract

RTG uses a direct-emitter intermediate representation. The shared lowering
code calls a small set of typed value, storage, control-flow, and call
operations; the selected backend emits the final machine or Wasm encoding as
each operation arrives. There is deliberately no materialized instruction
graph today.

This contract is an internal RTG ABI. It is not the platform C ABI, the Go ABI,
or an instruction-set register allocation policy. A backend may keep the
values in registers, locals, memory, or generated C expressions as long as the
observable behavior below is preserved.

## Value locations

The scalar emitter exposes three transient value locations:

| Location | Meaning |
| --- | --- |
| primary | Expression result, scalar return value, pointer in a string or slice |
| secondary | Second operand or temporary address, string length, slice length |
| tertiary | Third operand or index, slice capacity |

Operations name these locations directly, for example
`rtgAsmCopyPrimaryToSecondary`, `rtgAsmLoadPrimaryStack`, and
`rtgAsmAddSecondaryTertiary`. Architecture implementation functions may use
physical register names because those names describe their encoding.

Unless an operation or call result says otherwise, an emitter operation may
clobber all three transient locations. Values that must survive another
operation must be placed in a compiler stack slot, global storage, or on the
temporary value stack.

## Normalized values and storage

`rtgBackendValueSlotSize` is eight bytes. It is the normalized compiler stack
and aggregate layout unit, not the target pointer width or target `int` width.
Machine properties come from `rtgTargetProfile`.

- Scalars and pointers occupy one normalized slot when stored in a compiler
  frame or struct.
- A string is two words: pointer in primary, then length in secondary. Its
  stored form is pointer at offset 0 and length at offset 8.
- A slice is three words: pointer in primary, length in secondary, then
  capacity in tertiary. Its stored form uses offsets 0, 8, and 16.
- An interface is two words: payload at offset 0 and canonical dynamic-type
  tag at offset 8. A nil interface has both words zero. Boxing a typed nil
  pointer retains its nonzero type tag. Values larger than one slot are held
  in persistent storage and the payload word points at that copy. Assignment,
  calls, returns, and aggregate storage must preserve both words.
- Struct fields are laid out in declaration order at eight-byte-aligned
  offsets. Struct size is rounded to eight bytes.
- Tuples use the same aggregate layout as anonymous structs.
- Arrays are contiguous element storage. Their current call flattening uses
  the target native `int` width; this is a compatibility part of the current
  contract and must not be inferred from `rtgBackendValueSlotSize`.
- BSS/global and data references are section-relative offsets. The image
  backend owns their final addresses and relocations.

Compiler frame offsets identify normalized local storage. The temporary value
stack is LIFO and is also used to stage call words. Backends may implement that
stack using the hardware stack, Wasm locals plus linear memory, or an
equivalent private mechanism.

## Calls and returns

Arguments are flattened left to right into numbered call words. The caller
evaluates them in reverse and pushes their words so call word 0 is consumed
first. The first six words use backend-defined fast locations; remaining words
stay in caller-provided overflow storage. The caller removes overflow words
after the call.

| Value | Call words |
| --- | --- |
| scalar or pointer | one |
| string | pointer, length |
| slice | pointer, length, capacity |
| interface | payload, dynamic-type tag |
| struct or tuple | flattened storage words in field order |
| array | flattened native-width element words |

The current native mappings are implementation details, listed here to make
porting and compatibility review concrete:

| Backend | words 0 through 5 | overflow stride |
| --- | --- | --- |
| amd64 | RDI, RSI, RDX, RCX, R8, R9 | 8 bytes |
| 386 | EBX, ESI, EDX, ECX, EAX, EDI | 4 bytes |
| aarch64 | X3, X4, X1, X2, X5, X6 | 16 bytes |
| arm | R3, R4, R1, R2, R5, R6 | 4 bytes |
| wasm32 | virtual call locals 0 through 5 | virtual value stack |

A scalar returns in primary. Strings and slices return in the value locations
shown above. Structs and tuples use a hidden destination pointer in call word
0; explicit parameters begin at call word 1 and the callee writes the complete
aggregate before returning. Calls are internal RTG calls unless a dedicated OS
or foreign-call adapter says otherwise.

## Control flow and image responsibilities

The shared emitter allocates opaque labels and requests marks, conditional or
unconditional branches, and calls. A backend must support forward references
and reject unresolved or out-of-range references instead of silently emitting
a corrupt image.

The architecture emitter owns instruction selection and architecture
relocations. OS/image composition owns executable headers, sections, entry
code, imports, system-call adapters, and final relocation placement. Backend
IR operations must not select an operating system or executable format.

## Adding a backend

A new backend must:

1. Map primary, secondary, tertiary, scratch addresses, six fast call words,
   and overflow call words without assuming amd64 registers.
2. Preserve normalized stack, string, slice, aggregate, and hidden-result
   layouts.
3. Implement the shared storage, arithmetic, comparison, control-flow, call,
   and relocation operations or produce a clear unsupported diagnostic.
4. Keep architecture-local physical names behind the neutral dispatch API.
5. Pass the backend contract tests and compile the cross-target regression
   corpus before target-specific optimizations are added.

Introducing a materialized IR later is compatible with this design: the
existing direct-emitter operations become the first instruction vocabulary,
and the documented value/call/storage semantics remain the lowering boundary.
