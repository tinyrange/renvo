# Pinned Kbuild handshake

Milestone M1 uses upstream Linux 6.12.99 LTS, x86_64 `tinyconfig`. The tag,
peeled commit, and kernel.org archive digest are fixed in `pin.env`; changing
any of them is an explicit corpus refresh.

Run `./tools/kbuild/prepare`. It creates a fresh temporary tree, verifies the
archive, builds Renvo, keeps host utilities on `HOSTCC`, and assigns only target
`CC` to `renvo cc`. Success currently means:

- Kconfig identifies the bounded GCC-compatible driver and its external GNU
  assembler contract;
- unsupported option probes return failure;
- `scripts/mod/empty.c` becomes a valid ET_REL object and its dependency rule
  is consumed by `fixdep`;
- the build parses the closure's anonymous aggregates, enums, constant array
  bounds, and function-pointer declarators, then reaches the checked GNU
  attribute blocker reported at `scripts/mod/devicetable-offsets.c:152:19`.
- the M2 frontend preprocesses that complete translation unit, expands the
  checked `SIZE_`/`OFF_` paste results, and records its transitive dependency
  rule plus CPU/RSS telemetry.

The script leaves its tree, full log, timing/RSS telemetry, and read-only JSON
census in the printed workspace. CPU, RSS, and compiler bytes are observations,
not M4 gates.

After completing the same pinned tinyconfig once with the system compiler, run
the M4 semantic gate over every recorded target C command:

```sh
go build -o /tmp/renvo ./cmd/renvo
go run ./tools/kbuild/syntax \
  -kernel /path/to/linux-6.12.99 \
  -compiler /tmp/renvo \
  -expected 482
```

The system compiler is used only to preprocess each exact Kbuild command. Renvo
receives each result as a standard preprocessed-C `.i` input, then
performs the GNU C11 parse and type check sequentially while retaining at most
one translation unit at a time. The command prints its workspace so a failing
unit remains available for diagnosis. The frozen M4 result is 482/482 on the
fixed-point self-hosted compiler; the uninterrupted replay took 19m34.765s and
peaked at 141,024 KiB RSS across the driver and its children.

Regenerate a census for any prepared tree without modifying it:

```sh
go run ./tools/kbuild -kernel /path/to/linux -out dashboard.json
```
