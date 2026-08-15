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
census in the printed workspace. CPU and RSS are observations, not gates. The
checked frontend/backend binary-size gates remain authoritative.

Regenerate a census for any prepared tree without modifying it:

```sh
go run ./tools/kbuild -kernel /path/to/linux -out dashboard.json
```
