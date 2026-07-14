# Executable capability probes

Each `case_NNN` directory is a complete Go module used by the generated
frontend capability report. Human-owned labels, expectations, and commentary
live in `manifest.json`; compiler-owned status and messages live in
`results.json` and must be updated by running the probes.

Build a host frontend, backend, and stage3 compiler, then capture both stages:

```sh
mkdir -p sandbox/report-bin
go build -o sandbox/report-bin/rtg-backend .
go build -o sandbox/report-bin/rtg-host ./rtg/cmd/rtg
RTG_BACKEND="$PWD/sandbox/report-bin/rtg-backend" \
  RTG_STDROOT="$PWD/rtg/std" \
  sandbox/report-bin/rtg-host -t linux/amd64 -s \
  -o "$PWD/sandbox/report-bin/rtg-stage1" ./rtg/cmd/rtg
PWD="$PWD" sandbox/report-bin/rtg-stage1 -t linux/amd64 -s \
  -o "$PWD/sandbox/report-bin/rtg-stage2" ./rtg/cmd/rtg
PWD="$PWD" sandbox/report-bin/rtg-stage2 -t linux/amd64 -s \
  -o "$PWD/sandbox/report-bin/rtg-stage3" ./rtg/cmd/rtg
go run ./cmd/rtgreport -capture \
  -run host="$PWD/sandbox/report-bin/rtg-host" \
  -run stage3="$PWD/sandbox/report-bin/rtg-stage3" \
  -backend "$PWD/sandbox/report-bin/rtg-backend" \
  -stdroot "$PWD/rtg/std" -target linux/amd64
```

`go run ./cmd/rtgreport` regenerates only the HTML and corpus counts from the
checked capture. `go run ./cmd/rtgreport -check` verifies fixture completeness,
expected outcomes, the compiler/fixture source digest, and generated-file
drift. CI runs the latter command.
