# AArch64 CoreMark: guarded read-span loops

## Implementation

The AMD64 JIT now recognizes read-only affine memory streams of the form
`base + (uint32(index) << scale)` with an invariant stride and a low-word
loop recurrence. Recognition is based on IR semantics, not a benchmark name,
guest PC, or particular matrix dimensions.

Before entering the fast body it proves up to sixteen iterations of every
stream fit within one readable guest page and do not wrap their 32-bit index.
Each stream gets a private translated base. Failed admission returns to the
existing checked body without publishing a permission fact or reporting a
premature guest fault. Actual loads retain their original ordering.

The fast body uses scaled host loads, omits the per-load page guards, and only
captures cold-exit leaves at chunk boundaries before re-entering admission.
A combined span/budget check preserves the original exact instruction budget.
Iteration and retirement arithmetic still occurs per iteration; this is not
complete boundary-only accounting. AMD64 subtraction predicates also use CMP
on allocated operands rather than destructive copies and SUB.

Initial scope: AMD64 host, checked read-only loops of at most sixteen guest
instructions, at most four recognized streams, scales 2/4/8 and scalar loads
up to eight bytes. Stores, intermediate guards, unsupported recurrences, and
failed span proofs retain checked execution. ARM64 host and direct-memory mode
retain the existing loop emitter. Whole-span admission is conservative: it can
reject a span even when a shorter run would succeed.

## Actual emitted matrix loop

| Continuing iteration | Before | After |
| --- | ---: | ---: |
| Instructions | 41 | 20 |
| Bytes | 185 | 82 |

This is the ordinary continuing iteration inside an admitted chunk. Entry,
chunk boundaries, normal final exits and failed proofs cost extra. Instruction
counts are not micro-op counts or runtime speed ratios. The final compiled
matrix region is larger (2322 versus 1728 bytes) because it retains the checked
body alongside the guard and fast body. Both plain matrix multiplication and
the bit-extraction matrix variant emitted read-span fast bodies.

AT&T syntax, source before destination. Addresses below are offsets within
this run's JIT arena, not guest addresses.

```asm
1651c  mov    %r8d,%ecx  # A index (low 32 bits)
1651f  mov    -0x160(%rbp),%rsi  # prevalidated A translation + base
16526  rex movswl (%rsi,%rcx,2),%ebx  # signed A load; no per-load check
1652b  add    $0x1,%r8d  # advance guest A index
16532  mov    %r9d,%ecx  # B index (low 32 bits)
16535  mov    -0x168(%rbp),%rsi  # prevalidated B translation + base
1653c  movswl (%rsi,%rcx,2),%r12d  # signed B load; no per-load check
16541  mov    0x0(%r11),%rcx  # N
16545  add    %ecx,%r9d  # advance B index
16548  mov    %rbx,%r13
1654b  imul   %r12,%r13  # multiply
1654f  add    %r13d,%r10d  # accumulate
16552  add    $0x1,%r15  # exact completed-iteration count
16556  sub    $0x7,%r14  # exact guest-instruction charge
1655a  mov    0x40(%r11),%esi  # loop bound
1655e  cmp    %r8d,%esi  # compare without destructive copies
16561  je     0x1659d  # normal guest loop exit
16563  cmp    -0x170(%rbp),%r14  # combined budget/span admission
1656a  jb     0x1656e  # chunk or budget boundary
1656c  jmp    0x1651c  # continue proven fast loop
```

## Validation

- Complete `./internal/rfe/runtime` package passed.
- New `TestNativeLoopReadSpanSignedStateAndBoundaries` compares against an
  independent scalar oracle across all budgets 1..64, chunk-edge loop exits,
  first/second-load faults after a completed span, denied permissions,
  signed loads, 32-bit index wrap, high initial bits and noncanonical addresses.
  It checks complete guest state, flags, PC, retirement, memory accesses,
  loop counts, remaining budget, and fault address.
- Existing stride, spill, inherited-state, predicate and quantum tests passed.
- Both compiler tools built successfully.
- Canonical preflight passed in 26 seconds, unchanged 60-second limit.
- Unchanged 100,000-iteration CoreMark guest: both seed CRC sets validated;
  retired guest instructions remained 59,156,887,738; profile_errors=0.

## Indicative whole-benchmark measurement

Same fixed scored guest and profiling workflow; host Intel Core i7-13620H.
Timings are derived from the port's microsecond ticks, not its integer-rounded
Iterations/Sec field.

| Seed run | Earlier baseline | Final implementation |
| --- | ---: | ---: |
| Performance | 14.699443 s | 14.416754 s |
| Validation | 14.660919 s | 14.343884 s |
| Combined | 29.360362 s | 28.760638 s |

Combined elapsed time is about 2.0% lower. This is one before/after profiled
comparison, not a statistically established speedup or a QEMU comparison.
The much smaller inner body does not imply an equivalent whole-CoreMark gain.

No benchmark sources, build flags, resource limits, or existing tests were
weakened. Existing unrelated working-tree changes were preserved. Changes
remain local; no commit, push or PR was created.
