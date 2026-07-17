# RTG frontend

The release frontend is a standalone executable. It contains the RTG backend
and an embedded copy of the supported standard-library sources, so it can
compile a module without `RTG_BACKEND`, `RTG_STDROOT`, or files installed next
to the executable.

The frontend resolves `//go:embed` directives for `string`, `[]byte`, and
`embed.FS` package variables. Embedded file-system payloads are compressed in
the linked program, which is also how release frontends carry `rtg/std` without
a generated source bundle.

Bundling is optional for development builds. A Go-built bootstrap frontend
includes the standard library when built with the `rtg_bundle` tag:

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

Source selection honors Go filename suffixes, `//go:build`, and legacy
`// +build` constraints. Built-in tags describe the selected RTG target and
include `rtg`, `unix`, and the project aliases `wasi`, `aarch64`, and `wasm32`.
Custom tags, including an explicit `go1.x` release policy when needed, are
provided with `-tags`; RTG does not infer a Go toolchain release tag.

Module dependencies are resolved deterministically without network access.
The frontend understands `require`, `replace`, and `exclude` directives and
searches, in order, local replacement paths, the main module's `vendor` tree,
and the read-only Go-style module cache named by `RTG_MODCACHE`. Required
versions are selected across the on-disk module graph before a conflicting
earlier package collection is retained. A missing
module is reported as an offline dependency error; RTG does not fetch modules,
consult proxies, or use authentication credentials. Populate the cache or add
a local replacement before compiling.

For Windows applications, `-windows-gui` selects the GUI PE subsystem instead
of the default console subsystem. For example:

```sh
rtg -t windows/386 -windows-gui -o app.exe ./cmd/app
```

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
| Windows arm64 | `rtg-windows-arm64.exe` |
| macOS arm64 | `rtg-darwin-arm64` |
| WASI wasm32 | `rtg-wasi-wasm32.wasm` |

Each one is a single file and can compile any output target listed by `rtg`
when run without arguments. No repository checkout, adjacent standard library,
Go installation, or separate `rtgx` process is required.
