# Compiler JIT benchmarks

The compiler JIT benchmark suite measures the custom-backend path as separate
phases. It is diagnostic rather than a performance gate: the strict compiler
limits remain owned by `backend/main_test.go`.

Run it on a supported native host with:

```sh
go test -run '^$' -bench 'Benchmark(Definition|Prepare|Frontend|Prepared|Linked|Custom)' -benchmem ./internal/backendjit
```

For stable comparisons, use several samples and compare them with `benchstat`:

```sh
go test -run '^$' -bench 'Benchmark(Definition|Prepare|Frontend|Prepared|Linked|Custom)' -benchmem -count 10 ./internal/backendjit > old.txt
# make the change
go test -run '^$' -bench 'Benchmark(Definition|Prepare|Frontend|Prepared|Linked|Custom)' -benchmem -count 10 ./internal/backendjit > new.txt
benchstat old.txt new.txt
```

The benchmarks isolate these costs:

- `DefinitionPipeline`: parse imports, resolve the RTG definition, and generate
  the prepared-backend Go source.
- `PrepareCold`: the definition pipeline plus compiling and encoding a new
  host-native backend artifact.
- `PrepareCacheHit`: definition identity calculation and artifact validation
  when the compiled backend is already cached.
- `FrontendUnitForJITBackend`: frontend work needed to produce the target-bound
  unit consumed by a custom backend.
- `PreparedBackendCompile`: map and enter the prepared compiler, compile one
  unit, and collect its output. This includes the file protocol used by the
  current compiler boundary.
- `LinkedImageLoadAndRun`: map, bind, protect, enter, and release a minimal
  generated program without frontend or backend compilation.
- `CustomBackendEndToEnd`: frontend unit construction followed by the warm
  prepared-backend compile path.

Each benchmark reports allocations plus the relevant definition, unit,
artifact, native-image, and output byte counts. Collect native results on every
host being compared: Linux amd64 and arm64, Windows amd64, and Darwin arm64.
The ELF loader also cross-builds for Linux 386 and arm, and the PE loader for
Windows 386 and arm64; those architectures need native or emulated runners for
execution timings. The current checked-in Go bootstrap compiler itself does
not build as a 32-bit Go program because its generated source relies on
64-bit-sized `int` constants, so 386 loader coverage is compiled separately
from end-to-end bootstrap coverage. Cross-host timing results should not be
combined because MAP_JIT, VirtualAlloc, mmap, import binding, and filesystem
costs differ.
