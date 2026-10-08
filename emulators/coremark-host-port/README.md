# Freestanding Linux x86-64 comparison port

Original host integration for the same unchanged upstream revision, license
and source set described in `../coremark-port/README.md`. This directory does
not vendor upstream CoreMark. Copy these three files to
`sandbox/aarch64-coremark/host-port/` next to the provisioned upstream sources.
Use the approved `coremark.host_build(1000000)` and `coremark.host_run()`
workflows. Fixed scalar generic x86-64 flags and both seed sets are recorded in
`docs/aarch64-coremark-host-comparison.json`.

Only syscall bindings, startup and host identification differ from the AArch64
port. Both seed sets must validate and each timed interval must exceed ten
seconds. Rates use microsecond ticks rather than truncated integer seconds.
This is a separate host artifact, never a replacement for the scored guest ELF.
