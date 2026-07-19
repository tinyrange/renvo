# Renvo frontend test corpus

`frontend_tests` is an acceptance corpus kept outside the frontend packages so
it can survive a frontend rewrite. Each case is its own Go module directory and
must print only `PASS\n` on success.

- `quick/` contains 300 tests intended to run on every frontend check.
- `extended/` contains 2250 broader interaction tests gated by `RENVO_FRONTEND_EXTENDED_TESTS=1`.
- `regressions/` contains hand-maintained cases that are never replaced by the generator.
- `negative/` contains checked rejection cases with exact phase, code, source location, and message expectations.

`corpus_manifest.json` records case, declared-variant, and normalized AST-shape counts. Tests recompute those fingerprints from the checked tree, so clone count cannot stand in for structural coverage.

By default the harness validates that each corpus case is valid host Go and prints `PASS\n`. If `./cmd/renvo` exists, the harness builds it with host Go and also checks compiler output. Set `RENVO_FRONTEND=/path/to/compiler` to test a specific compiler, such as a stage2 self-hosted binary.

The generated corpus is maintained by:

```sh
go run ./frontend_tests/generate_tests.go
```
