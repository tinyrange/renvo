# Compiler performance gates

`internal/perfgate/policy.json` owns the thresholds, reference revision, sampling
settings, and required Tier 1 matrix. `cmd/renvoperf` is the shared local and CI
driver. Native and WASI targets use the complete bundled compiler (`cmd/renvo` with
`renvo_bundle`), including source discovery, frontend analysis and lowering,
the in-process backend, and writing the executable. It does not require the
backend to accept Go source directly.

## Policy

| Measurement | Required ceiling |
| --- | --- |
| CPU, median of three | Reference +25% |
| Peak memory, median of three | Reference +20% |
| Peak memory, every candidate sample | 256 MiB |
| Stripped stage-3 compiler, median of three | Reference +10% |
| Stripped stage-3 compiler, every candidate sample | 8 MiB |
| VM instructions, median of three | Reference +20% |

VM linear memory also has the 256 MiB absolute and +20% relative memory limits;
the host VM process is measured independently. Native and WASI self-hosting compilers use a 192 MiB arena by default. Windows
uses the explicitly requested 256 MiB arena. The Windows process-memory ceiling
remains 256 MiB; committed runtime overhead may therefore still fail that gate.
The VM prepared-backend workload uses the production preparation tool’s 96 MiB
compiler arena.

Larger increases block feature inclusion. The maintainer evaluates the value
of additional features and their cost case by case. There is no automatic
baseline ratchet, exception flag, or formal waiver procedure. Ordinary policy
changes remain visible in review; an absolute ceiling is never overridden by
a passing relative result.

## Workloads and reference

For native and WASI targets, both source revisions are bootstrapped with host Go to stage0, then compiled
through stage1 to a self-hosted stage2 for the selected target. Bootstrap work
is excluded from the measurements. Each stage2 then builds its own revision of
the complete compiler to stage3: one warm-up per revision, followed by three
samples per revision with alternating execution order. Each invocation starts
a fresh process. Filesystem caches may be warm; there is no compiler result
cache. Every generated stage3 must compile a small program that runs and prints
exactly `PASS\n`. This validation is outside the self-host measurement.

The reference is pinned to a real commit in the policy. Both revisions run
serially on the same machine with the same compiler flags, arena, and execution
engine. The default local command creates and removes a detached reference
worktree. CI checks out that exact revision separately. Measurements compare
like target/host/engine pairs, never values from different operating systems.
The initial reference is the pre-change main revision; timings are measured
afresh rather than taken from a fabricated or machine-specific timing file.

VM32 instead measures a prepared custom backend compiling a canonical compact
unit. It uses the existing `internal/backendjit/testdata/semantic_runtime.go`
program, covering structs, pointers, slices, branching, floating point, and
64-bit arithmetic. The same candidate fixture bytes are supplied to both
revisions. Each revision prepares its own VM-hosted backend and lowers the
fixture before timing; no backend Go-source parser participates in the measured
work. Warm-up and median sampling are unchanged. Each produced VM program must
execute and print `PASS\n`. The artifact gate measures the prepared compiler’s
bytecode, not the smaller output program. This avoids full compiler self-hosting
inside an interpreter while exercising its browser-backend role.

## Measurement definitions

CPU is user plus kernel CPU across all threads of the measured process.
Linux/macOS use process resource usage after exit. Windows starts the process
suspended, assigns it to a Job Object, resumes it, then reads the job's CPU and
memory high-water counters after exit. This prevents missing a short-lived
compiler's resource usage. The Windows definitions follow Microsoft's
[job CPU accounting](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_accounting_information)
and [job memory accounting](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information).

Linux reports `ru_maxrss` in KiB and Darwin in bytes; both are normalized to
bytes and labelled `peak_rss`. Windows reports `peak_job_commit`: the job's peak
committed private memory, **not RSS**. This is a stable high-water measurement
that remains available after exit. The same numerical policy applies, but
cross-OS memory numbers do not represent identical physical quantities.

The workload is one OS process with the backend in process. VM, QEMU, and
Wasmtime execution includes the engine's CPU and memory overhead; VM source
loading is included too. This is not a general subprocess-tree profiler.
Windows rejects unexpected child processes; Unix resource usage measures the
compiler process. If compiler execution starts delegating to subprocesses, the
measurement scope must be updated before that architecture change lands.

CPU accounting removes time spent descheduled from the main signal. It still
varies with CPU frequency, cache contention, and host load. Same-runner paired
samples and medians reduce that noise without promising perfectly deterministic
CI. Wall time remains in the report for diagnosis and a 15-minute per-invocation
hang timeout; it is not the performance threshold. VM also has an outer
500-million-instruction hang limit. Missing runners, zero measurements, failed
compilation, and failed smoke tests fail the gate rather than skipping it.

## Coverage and commands

PRs, merge-queue commits, and main each run all nine Tier 1 targets in separate
Linux, Windows, macOS, and virtual-target workflow runs. The workflows share
`performance-run.yml` and select their targets from the policy. Linux ARM uses QEMU user emulation; AArch64 runs
natively. WASI uses pinned Wasmtime. VM uses the candidate's Go-built VM engine
for both source revisions, with deterministic instruction counts as an
additional signal. Engine changes themselves need review because both sides
share the engine. The `Required` aggregate waits for successful completion of all four workflow
runs for the exact commit and event. Missing, failed, cancelled, and skipped
workflows cannot satisfy it.

```sh
./tools/check performance
RENVO_PERF_TARGET=vm/vm32 ./tools/check performance
RENVO_PERF_TARGET=wasi/wasm32 ./tools/check ci-performance
go run ./cmd/renvoperf -matrix
```

Both check modes enforce exactly the same policy. Native execution requires a
compatible host. QEMU requires Linux; WASI requires Wasmtime and can also be
checked locally on macOS. WASI uses one preopened repository workspace and
places temporary outputs in its ignored sandbox directory. The default
report is `sandbox/performance/report.json`. Set `RENVO_PERF_REPORT` to retain
separate runs and `RENVO_PERF_REFERENCE` to use an existing clean checkout of
the pinned reference. CI uploads reports even when a gate fails. Reports include
the policy, revisions, host, engine version where applicable, all samples, and
failure reasons, including CPU and memory from failed compiler invocations.
`./tools/check full` runs the selected target's gate with the
full functional suites; only CI spans every required host.

The old direct-backend elapsed-time, calibration, and split frontend/VM gates
are superseded by this workload. Functional corpus instruction/memory safety
limits remain. The Node WASI profiler is diagnostic telemetry; `--check` directs
users to this shared gate.

## Superseded gates

These are the limits in the pre-change implementation (some older README
numbers were stale). The new full-compiler workload replaces these smaller,
separate workloads; passing it does not establish that the old limits still pass.

| Old check | Removed or replaced constraint |
| --- | --- |
| `TestCompilerPerformance` | Best-of-three 50 ms wall time (Darwin 175 ms), 16 MiB RSS, 320 KiB artifact (Windows/386 324 KiB; Darwin 640 KiB) |
| `TestCompilerResourceGates` | Separate 16 MiB RSS and 320 KiB artifact checks (ARM 322 KiB; Windows/386 324 KiB) |
| `TestCompilerPerformanceWASI` | 150 ms wall time, 20 MiB RSS, 384 KiB artifact |
| `TestFrontendCompilerPerformance` | Separate 64 MiB RSS and 4 MiB stage-3 artifact; synthetic CPU calibration was telemetry and is removed |
| VM backend performance assertions | 2 MiB bytecode, 400,000 instructions, 80 MiB VM memory for the small backend workload |
| VM frontend performance assertions | 6.5 MiB bytecode, 4 MiB + 10 KiB output, 150 MiB memory, and a redundant 15-billion-instruction assertion; the functional runner's 15-billion-instruction limit remains |
| Frontend payload checks | 4 MiB + 10 KiB becomes the shared 8 MiB limit |
| Bundled frontend writable ELF segment | 166 MiB becomes the shared 256 MiB limit; this static segment check is separate from measured peak memory |
| Node WASI profiler `--check` | 2 MiB frontend and 1-second wall-time check |

Compiler system profiles also move from 160 MiB to a 192 MiB arena, and their
4 MiB + 10 KiB / 6 MiB binary limits become 8 MiB. Ordinary functional tests,
VM corpus instruction/memory limits, and arena-exhaustion correctness checks
remain in place.
