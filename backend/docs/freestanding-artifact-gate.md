# Freestanding artifact gate

`renvoboard` validates a linked ELF image against an object-target plus board
composition before the image is flashed. The gate reads the ELF class, byte
order, machine, ABI flags, allocated sections, and unresolved symbols from the
actual linker output, then applies the composition's placement, entry, vector,
import, flash, static RAM, heap, stack, guard, and reserved-memory constraints.

For the initial CH32V003 profile:

```sh
go run ./backend/cmd/renvoboard -board ch32v003 -vector renvo_vectors \
  -heap 128 -stack 512 firmware.elf
```

Successful and failed validations both print deterministic per-section and
aggregate resource accounting. A failed validation also prints every
localized violation and exits nonzero, so the command can be placed directly
between linking and flashing in a board pipeline.

The command consumes only a final linked artifact. Startup, vector generation,
linker-script selection, and instruction emission remain separate from this
host-side board gate.
