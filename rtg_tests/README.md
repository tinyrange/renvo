# RTG Frontend Test Corpus

`rtg_tests` is a frontend acceptance corpus kept outside `rtg/` so it can
survive a frontend rewrite.

Each case is its own Go module directory and should print only `PASS\n` on
success. The layout is intentionally close to the backend `tests/` corpus, but
allows package graphs and multiple files per case.

- `quick/` contains 250 tests intended to run on every frontend check.
- `extended/` contains 2250 broader interaction tests gated by
  `RTG_FRONTEND_EXTENDED_TESTS=1`.

By default the harness validates that each case is valid host Go and prints
`PASS\n`. If `./rtg/cmd/rtg` exists, the harness builds it with host Go and
also checks compiler output. Set `RTG_FRONTEND=/path/to/compiler` to test a
specific compiler, such as a stage2 self-hosted binary.

The generated corpus is maintained by:

```sh
go run ./rtg_tests/generate_tests.go
```
