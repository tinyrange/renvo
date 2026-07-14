# Omnibus result ABI

The backend omnibus publishes one authoritative 64-byte memory block. It does
not require stdout, files, process exit, semihosting, UART, or a peripheral.
Optional transports only copy or display this block.

`omnibus/resultabi/protocol.json` is the protocol source of truth. Running
`go generate ./omnibus/resultabi` produces both the Go constants and the strict
C89 header. All multi-byte fields are little-endian even on a big-endian
target. The exported symbol is the deliberately short `rtgres`; firmware must
place it in `.rtg.result` and retain it in the linked symbol table.

## Commit ordering

Writers initialize magic, version, size, and profile before state. Before each
probe they publish `current_probe`, increment `sequence`, and only then publish
`running`. A successful comparison updates `completed_probes` and `signature`.
A failed comparison writes `failure_probe`, `expected`, and `observed` before
committing `failed_comparison`. Passing writes the final signature before
committing `passed`.

State is the commit field. Target implementations must use volatile stores or
equivalent compiler barriers for the result block and preserve the order
above. A debugger that stops a trapped or hung target can therefore identify
the last-entered probe even when no failure handler ran. `sequence` helps a
live reader detect that it sampled while the writer was changing fields: read
it, copy the block, then reject/retry if a second read differs.

Success requires all three of:

- state `passed`;
- the requested target/profile identifier;
- the independently recorded profile-specific final signature.

A passed flag alone is never accepted.

## Debugger-only collection

The host decoder finds `rtgres` from the linked ELF rather than assigning it a
universal RAM address. Capture any raw memory range containing that address and
tell the decoder the address corresponding to byte zero. For example, with a
CH32V003 OpenOCD/GDB setup and no UART or semihosting:

```sh
riscv-none-elf-nm firmware.elf | grep ' rtgres$'
openocd -f interface/wlink.cfg -f target/wch-riscv.cfg \
  -c 'init; halt; dump_image ram.bin 0x20000000 2048; shutdown'
go run ./cmd/rtgresult \
  -artifact firmware.elf -memory ram.bin -base 0x20000000 \
  -expected-profile 0x00320001 -expected-signature 0x0123456789abcdef \
  -hosted
```

On validated success, hosted mode emits exactly `PASS\n`. On failure it emits
no stdout and reports the state, last probe, or comparison mismatch on stderr.
Simulator and hardware adapters should preserve the same block unchanged.
