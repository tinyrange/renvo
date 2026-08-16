# Pinned Kbuild handshake

Milestone M1 uses upstream Linux 6.12.99 LTS, x86_64 `tinyconfig`. The tag,
peeled commit, and kernel.org archive digest are fixed in `pin.env`; changing
any of them is an explicit corpus refresh.

Run `./tools/kbuild/prepare`. It creates a fresh temporary tree, verifies the
archive, builds Renvo, keeps host utilities on `HOSTCC`, and assigns only target
`CC` to `renvo cc`. Set `RENVO_LINUX_ARCHIVE` to a previously downloaded pinned
tarball to skip the network transfer; the SHA-256 check still runs. Success
currently means:

- Kconfig identifies the bounded GCC-compatible driver and its external GNU
  assembler contract;
- unsupported option probes return failure;
- `scripts/mod/empty.c` becomes a valid ET_REL object and its dependency rule
  is consumed by `fixdep`;
- the build parses the closure's anonymous aggregates, enums, constant array
  bounds, function-pointer declarators, and GNU attributes, then reaches the
  explicit `devicetable-offsets.c` assembly/code-generation frontier;
- the M2 frontend preprocesses that complete translation unit, expands the
  checked `SIZE_`/`OFF_` paste results, and records its transitive dependency
  rule plus CPU/RSS telemetry.
- the M5 gate sends a synthetic leaf through the pinned kernel's real
  `scripts/Makefile.build`, consumes Renvo's dependency rule with `fixdep`, and
  archives the resulting object in Kbuild's thin `built-in.a`; `readelf` checks
  its named sections, visibility, weak alias, and external-call relocation.

M6's first accepted upstream unit has a separate opt-in gate. Start with a
completed system-compiler build of the pinned tinyconfig, then run:

```sh
./tools/kbuild/first-unit /path/to/linux-6.12.99
```

The gate compiles the unmodified upstream `lib/union_find.c` with Renvo and its
real generated/forced kernel headers, compares its exported symbols and records
section/relocation telemetry against the reference object, runs the checked
SysV aggregate/algorithm harness, and requires a warning-free per-object
`objtool` pass. Because tinyconfig archives this optional routine without
referencing it, the gate uses `ld -u uf_find` to select exactly that member from
the reference build's thin `lib/lib.a`. It then performs the pinned `vmlinux.o`
archive-group link and warning-free link-stage `objtool --ibt` pass. The
reference object is restored even on failure, and the printed workspace keeps
all evidence. Set `RENVO_FRONTEND` to exercise a particular host or fixed-point
compiler binary.

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
performs the GNU C11 parse and type check through a bounded worker pool (the
logical CPU count by default, configurable with `-j`). Each worker retains at
most one translation unit. The command prints its workspace so a failing unit
remains available for diagnosis. The frozen M4 result is 482/482 on the
fixed-point self-hosted compiler: 16 workers completed the uninterrupted replay
in 17.008s (245.63s user / 17.85s system), with a 158,176 KiB maximum child RSS.
The 16-worker per-child upper bound is approximately 2.42 GiB; CPU and memory
remain telemetry rather than acceptance limits.

Regenerate a census for any prepared tree without modifying it:

```sh
go run ./tools/kbuild -kernel /path/to/linux -out dashboard.json
```
