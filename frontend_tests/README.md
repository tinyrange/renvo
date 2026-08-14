# Renvo frontend test corpus

`frontend_tests` is an acceptance corpus kept outside the frontend packages so
it can survive a frontend rewrite. Each case is its own Go module directory and
must print only `PASS\n` on success.

- `quick/` contains 300 focused tests.
- `extended/` contains 2250 broader interaction tests.
- `regressions/` contains hand-maintained cases that are never replaced by the generator.
- `negative/` contains checked rejection cases with exact phase, code, source location, and message expectations.

The normal `go test ./frontend_tests` command runs both corpus tiers, the
bundled frontend checks, and the full self-hosted frontend coverage.

Size accounting keeps the compiler plus native backends under 2 MiB and the
complete offline `std/`, `forms/`, and `device/` source bundle under 4 MiB.
Framework and device packages remain covered by standalone offline builds.

`corpus_manifest.json` records case, declared-variant, and normalized AST-shape counts. Tests recompute those fingerprints from the checked tree, so clone count cannot stand in for structural coverage.

Each positive module has a checked-in `expected.txt`. The normal harness
compares Renvo output directly with that value, avoiding thousands of host-Go
builds and their Go-cache footprint. It builds `./cmd/renvobootstrap` with Go
for stage0 coverage, then checks the embedded-backend self-hosted frontend
stages. Set
`RENVO_FRONTEND=/path/to/compiler` to test a specific compiler, such as a
stage2 self-hosted binary.

Use a focused regular expression while iterating:

```sh
./tools/check frontend 'map_frontend_lowering'
```

Add missing positive expectations with
`go run ./cmd/renvoexpect -write`. Negative cases continue to use their exact
diagnostic `expect.json` files.

The generated corpus is maintained by:

```sh
go run ./frontend_tests/generate_tests.go
```
