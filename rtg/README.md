# RTG frontend

The release frontend is a standalone executable. It contains the RTG backend
and a generated copy of the supported standard-library sources, so it can
compile a module without `RTG_BACKEND`, `RTG_STDROOT`, or files installed next
to the executable.

Bundling is optional for development builds. Generate the source bundle with:

```sh
go generate ./rtg/internal/driver
```

A Go-built bootstrap frontend includes the standard library when built with
the `rtg_bundle` tag:

```sh
go build -tags rtg_bundle -o rtg-stage0 ./rtg/cmd/rtg
```

That bootstrap still delegates code generation through `RTG_BACKEND`. Produce
a fully standalone RTG-built frontend by selecting the same tag while
self-hosting:

```sh
RTG_BACKEND=/path/to/rtgx ./rtg-stage0 \
  -tags rtg_bundle -t linux/amd64 -s -o rtg ./rtg/cmd/rtg
```

The resulting `rtg` can emit every supported target. Setting `RTG_STDROOT`
explicitly overrides the embedded library, which is useful when developing or
testing standard-library changes.

Release builds are published for every host currently supported by the RTG
backend:

| Host | Release executable |
| --- | --- |
| Linux amd64 | `rtg-linux-amd64` |
| Linux 386 | `rtg-linux-386` |
| Linux arm64 | `rtg-linux-arm64` |
| Linux ARM | `rtg-linux-arm` |
| Windows amd64 | `rtg-windows-amd64.exe` |
| Windows 386 | `rtg-windows-386.exe` |
| macOS arm64 | `rtg-darwin-arm64` |
| WASI wasm32 | `rtg-wasi-wasm32.wasm` |

Each one is a single file and can compile any output target listed by `rtg`
when run without arguments. No repository checkout, adjacent standard library,
Go installation, or separate `rtgx` process is required.
