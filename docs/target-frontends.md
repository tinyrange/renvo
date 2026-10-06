# Typed target operations and assembler frontends

A custom frontend should consume a selected backend's **operation vocabulary**,
not invent emitter calls, recognize kernel-specific instruction patterns, or
substitute runtime helper calls for assembly. The first implementation provides
**whole-function, explicit physical machine blocks**. Compiler-managed semantic
intrinsics and C inline-assembly constraints are separate, future integration
work; this interface does not pretend they are physical blocks.

## Backend-owned contract

An architecture advertises operations in `frontend_operations`. Public names
are independent of private encoder names and virtual-package qualification:

```text
frontend_operations {
    resources = [flags]
    move_immediate(destination:register, value:int64) {
        lower = sequence machineMoveImmediate
        reads = []
        writes = [destination]
        clobbers = []
        memory = none
        control = none
        ordering = ordered
        requires = []
    }
}
```

`lower` binds a backend Go encoder or bounded sequence with an emitter first,
then the declared operand types, and exactly the declared result type. It is
not a function in the emitted application. Encoders, instruction selection,
and relocation policy remain backend-owned. `extend arch` may add operations
and sequences but cannot replace an existing public operation.

The contract types currently supported are `register`, `condition`, `label`,
`address`, `int`, `int64`, `uint64`, `byte`, and `bool`. An optional `-> kind`
result is bound once and referenced by later operations in the same block.
The result is an encoder-time value or handle, **not a compiler-managed runtime
value**. Label results must be fresh handles; label operands must explicitly
state `defines = [operand]` or `references = [operand]`. A referenced label must
be bound once within its block. Results cannot be discarded or left unused.

Required properties are `lower`, `reads`, `writes`, `clobbers`, `memory`,
`control`, `ordering`, and `requires`. Location effects name operands, physical
registers, or backend-declared resources. Memory effects are `none`, `read`,
`write`, `read_write`, or `unknown`; control effects are `none`, `label`, `jump`,
`branch`, `call`, or `return`. All physical operations are `ordered`. These
properties are authored backend contracts, not automatically inferred proofs
about encoder bodies. No optimizer currently uses them to reorder blocks.
The frontend must account for the advertised fixed-register effects.

`requires` lists target descriptor capabilities. An operation is available
only when all of its requirements are present. Unsupported names and types
fail closed before encoder evaluation. Integer transport `int` is signed
32-bit, matching the portable evaluator, not the target's native integer width.
For stricter encoding limits use:

```text
immediates {
    value = bits 16
    offset = signed 12
}
```

`bits N` accepts signed or unsigned N-bit patterns, while `signed N` and
`unsigned N` use their usual ranges. Limits apply to literal operands; a
computed encoder result cannot bypass a literal-range requirement.

## Typed assembler source

Version 1 `.rtgasm` remains the existing expert, backend-helper sequence
interface. Version 2 is the restricted typed assembler frontend:

```text
rtgasm 2
assembly {
    answer(out:emitter) {
        let done = new_label()
        jump(done)
        move_immediate(register(rax), int64(99))
        bind_label(done)
        move_immediate(register(rax), int64(42))
        return()
    }
}
```

Supply this beside a bodyless Go declaration with the same name:

```go
package main

func answer() int
func appMain() int { return answer() - 42 }
```

For x86-64, compile the package with
`renvo -backend backend/definitions/linux_amd64.rtg -t linux/amd64 -o answer .`.
Use the actual selected backend's contract: the same operation name may have
a different operand type or effect contract on another architecture.

Operations and operands can span lines; separate instructions with newlines
or semicolons. Operands are typed literals or preceding result names. There
are no arbitrary expressions, method calls, casts, raw bytes, nested blocks,
or implicit emitter arguments in version 2. Registers and conditions must
belong to the selected architecture. Labels and addresses cannot be forged
from integers: obtain them from declared operations. For example, x86-64 and
RV32 support `let mem = address(register(...), int(...))` followed by `load`
and `store` on that address handle.

Blocks preserve source instruction order and backend-declared effects. For
example, x86-64 physical `move_immediate` emits MOV rather than the ordinary
compiler's XOR/push/pop shortcuts, preserving flags and avoiding a hidden
stack access. RV32 `compare` explicitly advertises writes to `t3` and `t4`;
it is a backend-defined comparison operation, not an assertion that RV32 has
an x86 flags register.

A block replaces an entire function. Its author is responsible for the selected
backend's calling convention, argument/result locations, register preservation,
stack balance, and executable control flow. `out:emitter` is only the source
wrapper, not an application parameter. This interface does not allocate
registers, insert prologues/epilogues, or check an application's memory accesses.

## Programmatic frontends

The ordinary-Go `renvo.dev/driver` API exposes:

- `ReadTargetVocabulary(source, filename, target, loader)` to resolve the selected
  contract. The import loader is a caller-owned capability, or nil for a closed
  standalone definition. The returned vocabulary includes target metadata,
  physical registers, conditions, resources, signatures, effects, and limits.
- `TargetBlock`, `TargetInstruction`, and `TargetOperand` to construct ordered,
  typed operations. `TargetOperand{Kind: "value", Value: name}` refers to a
  preceding result; other kinds specify literal data.
- `EncodeTargetAssembly(vocabulary, blocks, filename)` to validate and serialize
  version 2 source. Put it in the caller's `SourceFS` beside function declarations
  and compile with a selected CompilerJIT backend.

For example, after reading an x86-64 vocabulary:

```go
blocks := []driver.TargetBlock{{
    Name: "answer",
    Instructions: []driver.TargetInstruction{
        {Operation: "move_immediate", Operands: []driver.TargetOperand{
            {Kind: "register", Value: "rax"},
            {Kind: "int64", Value: "42"},
        }},
        {Operation: "return"},
    },
}}
source, diagnostics := driver.EncodeTargetAssembly(vocabulary, blocks, "answer.rtgasm")
```

The compact unit retains this source through ordinary linking, using the
existing assembly transport without changing the unit schema. CompilerJIT
validates it again against the selected definition and lowers it through the
existing bounded encoder evaluator. A prepared `.rtgb` retains the vocabulary
and definitions. Cached code is keyed by definition identity, source, and entry;
supplied materialized bytes do not bypass typed-source validation or emission.

## Current scope and deliberate exclusions

The checked-in starting vocabularies cover x86-64, RV32, and the MS-DOS 8086
backend: labels, moves, addition, comparison, branches, and return. X86-64 and
RV32 additionally advertise native-width base-plus-displacement loads/stores.
This is extensible vocabulary, not a claim of complete ISA coverage.

GAS/Intel source syntax, assembler directives, external symbols/data sections,
C extended-asm operands and constraints, tied/early-clobber operands, `asm goto`,
and compiler-managed semantic intrinsics are **not implemented by this change**.
Existing C template lowering is unchanged, not silently routed through a
whole-function assembly substitute. Those frontends must translate their own
semantics to appropriate contracts or report unsupported input. The current
materializer and programmatic assembler API use the ordinary-Go CompilerJIT
path; this does not claim self-hosted typed assembler evaluation.
