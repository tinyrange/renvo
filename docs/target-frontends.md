# Typed target operations and assembler frontends

A custom frontend should consume a selected backend's **operation vocabulary**,
not invent emitter calls, recognize kernel-specific instruction patterns, or
substitute runtime helper calls for assembly. The interfaces distinguish
**whole-function physical blocks**, conventional assembler spelling,
**compiler-managed runtime intrinsics**, and **C extended assembly**. These
share backend-owned contracts, but do not conflate their calling conventions.

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

The portable `renvo.dev/targetfrontend` API exposes the following contracts.
`renvo.dev/driver` retains compatibility wrappers for ordinary-Go callers:

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
blocks := []targetfrontend.TargetBlock{{
    Name: "answer",
    Instructions: []targetfrontend.TargetInstruction{
        {Operation: "move_immediate", Operands: []targetfrontend.TargetOperand{
            {Kind: "register", Value: "rax"},
            {Kind: "int64", Value: "42"},
        }},
        {Operation: "return"},
    },
}}
source, diagnostics := targetfrontend.EncodeTargetAssembly(vocabulary, blocks, "answer.rtgasm")
```

The compact unit retains this source through ordinary linking, using the
existing assembly transport without changing the unit schema. CompilerJIT
validates it again against the selected definition and lowers it through the
existing bounded encoder evaluator. A prepared `.rtgb` retains the vocabulary
and definitions. Cached code is keyed by definition identity, source, and entry;
supplied materialized bytes do not bypass typed-source validation or emission.

## Conventional assembler

Package discovery recognizes target-filtered `.s` files alongside `.rtgasm`.
Declare exported functions with `.globl` or `.global`; each definition binds a
bodyless declaration in the same package. Definition order, not declaration
order, determines entry indices. Multiple functions per file are supported.

```asm
.text
.globl answer
.type answer,@function
answer:
    movl $40,%eax
    movl $2,%edi
    addl %edi,%eax
    ret
.size answer,.-answer
```

`frontend_syntax` advertises dialects, mnemonic forms, typed operands, and
operand permutations. No architecture/opcode dispatch table lives in the
frontend. X86-64 and 8086 advertise AT&T and Intel forms; RV32 advertises native
forms. A blank parser syntax chooses the first advertised dialect. The
`.intel_syntax noprefix` and `.att_syntax prefix` directives select an advertised
style. `.text`, function `.type`, and `.size name,.-name` are
checked rather than silently discarded. Named local labels and nearest repeated
numeric `1f`/`1b` labels are scoped to a function. Instruction order, integer
ranges, and backend-defined register effects survive lowering.

`ParseTargetAssembler` parses a file; `ParseTargetInstructionBlock` parses an
inline body. Both return the same typed machine IR as programmatic frontends.
Address operands currently use base-plus-displacement, such as `8(%rax)` or
`[rax+8]`; scaled/indexed operands are rejected.

X86-64 advertises 8/16/32/64-bit MOV, ADD, and CMP spellings. `frontend_widths`
maps aliases to canonical physical registers, distinguishes operand widths,
and selects signed/unsigned preload and capture operations. Width aliases do
not create new physical registers: e.g. `eax` still has `rax` effects. Mixed
or incorrect spellings (such as `movq %eax,%rax`) are rejected. Backend metadata,
not frontend mnemonic suffix guesses, decides the instruction width.

## Managed bodies and semantic intrinsics

Version 3 assembly adds a compiler-owned native-word boundary:

```text
rtgasm 3
assembly {
    sum(out:emitter) {
        inputs = 2
        intrinsic = word_add
        yield()
    }
}
```

Declare `func sum(left, right int) int` beside this source. `frontend_intrinsics`
selects a private encoder with runtime `word` parameters, one word result, and
explicit effects. The frontend supplies no registers or encoder calls.
X86-64 and RV32 advertise `word_add` and `word_sub`; direct calls emit the
allocated body inline, not a runtime helper CALL. Function values and deferred
calls retain an ordinary callable wrapper. Arguments are evaluated once and
ordinary compiler spill semantics protect live values across the boundary.

`EncodeIntrinsicAssembly` serializes this semantic interface. For explicit
physical bodies at the same compiler boundary, use `ManagedBlock` and
`EncodeManagedAssembly`, or write version 3 operations with `argument(N)` and
one final `yield(register(...))` / `yield()`. Logical argument indices use source
parameter order. The backend's incoming call-word metadata is in **pop order**,
so parameter `i` maps to word `inputs-1-i`; the frontend never assumes the first
incoming register is the first parameter. Capture into the primary result and
the wrapper's return are compiler-owned.

`frontend_managed` advertises only caller-owned registers/resources and the
backend operations required for capture. Managed bodies cannot own the stack,
frame, or callee-preserved registers, or embed user CALL/RET sequences. The
current boundary supports zero through six scalar native words and zero or one
result; narrower Go values, aggregates, multiple results, and specialized custom
call adapters are rejected. Semantic pointer intrinsics and arbitrary SSA
intrinsic graphs are not advertised: current semantic blocks select one word
intrinsic, composable through ordinary frontend expressions.

## C extended assembly

With a selected `-backend`, C translation uses an authoritative assembly
compiler capability rather than pattern recognition. It supports `asm`,
`__asm__`, `volatile`, and `asm goto` with:

- backend-owned register classes (`r` and advertised fixed classes), `m`, and
  compile-time integer `i`/`n` inputs;
- `=` and `+` outputs, early-clobber `&`, and numbered/named matching inputs;
- named/numbered template operands, `%%`, unique `%=`, constant `%c`, and goto
  `%l` references (including the implicit input numbering of `+` outputs);
- backend-advertised register width modifiers (`%b`, `%w`, `%k`, `%q` on x86-64);
- declared register/resource clobbers, `cc`, and ordered `memory` barriers.

```c
int value = 20;
__asm__ volatile("addl %[input],%[result]"
    : [result] "+r"(value)
    : [input] "r"((int)22)
    : "cc");
```

C expressions and output lvalue addresses are evaluated once in ordinary
frontend code. A bounded backtracking allocator handles fixed classes and tied
or interfering values. Runtime words are placed with a parallel-copy scheduler;
cycles require a scratch register. Register output addresses are retained in
unused caller-owned registers. Read/write outputs load through those pointers;
outputs store using their original width, without overwriting adjacent storage.
X86-64 supports C 8/16/32/64-bit integer operands and native-width pointer
carriers, with signed read/write preloads. Other backends must explicitly advertise narrow helpers.

Goto exits capture outputs on both taken and fallthrough paths and return a
compiler-owned selector to the C control-flow lowering. No user branch may
escape to an unbound application address. Undeclared register writes/reads,
flag destruction without a resource clobber, CALL/RET, and preserved-register
clobbers fail closed. Alternative/combined constraint classes, high-byte
register modifiers, floating-point/vector operands, excessive runtime-word
pressure, and unsupported instruction forms produce diagnostics. The generic
selected-backend path never falls back to C approximations on failure. Legacy
C compilation without a selected assembly capability keeps its existing path.

## Portable/self-hosted materialization

`targetfrontend.MaterializeAssembly(unit, definition, filename, target, loader,
compiler, memory)` materializes linked source-preserving assembly without host
filesystem or process APIs. The import loader is caller-owned; the compiler
capability receives a **closed generated encoder program** and returns raw
`vm/vm32` bytecode. It may run in a separate arena/process. The materializer
executes it in a VM with a hard **500 million step / 96 MiB** ceiling; `memory=0`
selects that ceiling, and a nonzero value may impose a smaller bound.

Every source and binding is validated before compilation. Supplied code bytes
are not trusted. The service generates only reachable backend encoders and
emitter support, not an entire compiler/kernel. It applies the selected
backend's local relocation finalizer, checks completion, and validates exact
nonempty length-framed output before replacing the optional assembly child.
The original sources and function/source/entry identities remain in the unit.
The complete child stream is checked for duplicate tables, invalid portable
varints, and truncated/trailing input. The ordinary-Go CompilerJIT path retains
its source/definition/entry cache, with typed validation before cache lookup.

The self-host acceptance test compiles this public API with Renvo, generates an
evaluator, compiles the actual generated program with the self-hosted compiler,
executes its bytecode through the portable materializer, and checks the emitted
fragment. It does not supply prebuilt machine code or raise the harness's arena
budget. This API is an embedding boundary, not a claim that the bundled command
can infer an arbitrary selected definition or evaluator compiler automatically.

## Supported subset

The checked-in vocabularies cover x86-64, RV32, and MS-DOS 8086: labels, moves,
addition, comparison, branches, and return; x86-64 and RV32 additionally expose
base-plus-displacement memory operations. Available C/managed operations depend
on the selected target's advertised policy. This is extensible vocabulary, not
complete ISA, GAS, Intel, or GCC constraint compatibility.

External symbols, inter-function assembly relocations, data/rodata/BSS sections,
raw data directives, arbitrary symbol expressions, and unadvertised instruction
forms are rejected. Local relocation semantics remain backend-owned. As with
any explicit assembly, effect declarations are trusted backend contracts, not
proofs of encoder behavior or application memory safety.
