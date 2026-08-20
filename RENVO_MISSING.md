# Renvo support required by staragent

This document records the gaps between the Go standard-library surface used by
`j5.nz/staragent` and the Renvo tree in this checkout. It is a static inventory
based on `GO_STDLIB.md`, staragent's current source, and Renvo's `std/` tree and
documentation.

## Scope and conclusion

Staragent does not use either of Renvo's two explicitly excluded frontend
features: **generics** or **cgo**. Its ordinary Go language usage is therefore
within Renvo's stated frontend scope.

The immediate blockers are standard-library and hosted-runtime facilities. A
production staragent build imports 29 standard-library paths. Renvo has only a
subset of those packages, and package presence does not guarantee that the
needed API and semantics are complete.

The most difficult functional dependency is not JSON or formatting but a real
HTTPS client stack: staragent must connect to OpenAI/ChatGPT, validate TLS,
stream server-sent events, and support cancellation.

## Verified first compiler failure

Running the bundled Renvo bootstrap against the staragent package for
`linux/amd64` currently stops at the loader before type checking:

```text
agent.go:4:3: error RENVO-LOAD-020 (loader):
standard library package context is not included in this RENVO build
```

This confirms `context` as the first dependency blocker in source/import order.
The compiler reports one loader failure at a time, so the remaining entries
below are still based on the complete import/tree audit and will become visible
progressively as packages are added.

## Implementation progress checklist

This checklist distinguishes source implementation from validation. An item is
checked only for the exact surface described on that line; package presence does
not mean the complete Go API is supported.

### Implemented source surfaces

- [x] `encoding/hex`: byte encode/decode, string helpers, length helpers, and
  invalid-byte errors.
  - [x] Host-Go package tests cover known encoding and round trips.
  - [x] `frontend_tests/regressions/std_json_dynamic` runs through Renvo and
    the compiled executable prints `PASS\n`.
- [x] `encoding/base64`: standard and URL alphabets, padded and raw encodings,
  byte encode/decode, and string helpers.
  - [x] Host-Go package tests cover padded standard and raw URL round trips.
  - [x] The focused Renvo regression covers the unpadded URL encoding path.
  - [ ] Add malformed-input cases to the compiled regression.
- [x] `encoding/json` syntax validation and `RawMessage` support.
  - [x] Host-Go tests cover nested syntax, malformed syntax, escapes including
    `\uXXXX`, and `RawMessage` validation/copying.
  - [x] Execute syntax validation and `RawMessage` in a Renvo-compiled program.
- [x] `encoding/json` dynamic marshaling for `nil`, booleans, strings, signed
  and unsigned integers, `[]any`, `map[string]any`, and custom `Marshaler`
  values.
  - [x] Map keys are sorted for deterministic output.
  - [x] Host-Go tests cover nested dynamic trees and control-character escaping.
  - [x] Execute dynamic map/interface dispatch through Renvo; this specifically
    needs to exercise map iteration, interface type switches, recursion, and
    sorted keys in generated code.
- [x] Initial `encoding/json.MarshalIndent` for the supported dynamic values.
  - [x] Host-Go output test covers nested object/array indentation.
  - [ ] Compare more empty/nested values and prefix behavior with Go.
  - [x] Execute the indentation path through Renvo.
- [x] Added the positive frontend regression
  `frontend_tests/regressions/std_json_dynamic`, with checked-in `PASS\n`
  expectation, plus a focused test entry point.
  - [x] The focused regression passes under the Renvo compiler.

### JSON work still required

- [x] Decode JSON values into `any`, dynamic maps/slices, strings, booleans,
  integer numbers, and null, including escaped Unicode strings.
- [ ] Marshal and unmarshal arbitrary structs, including exported-field rules,
  embedded fields, JSON tags, `omitempty`, and field-name matching.
  - [x] Marshaling supports exported and embedded fields, JSON field names,
    `-`, `omitempty`, `,string`, pointers, and typed slices/arrays.
  - [x] Struct unmarshaling supports exported value fields, value-embedded
    structs, JSON field names, `-`, `,string` integers, and typed `[]int`.
  - [ ] Nil embedded pointers, arbitrary typed containers, case-insensitive
    matching, and Go's full dominant-field conflict rules remain.
- [ ] Support typed pointers, maps, slices, arrays, interfaces, and numeric
  destinations, with Go-compatible overflow and type errors.
- [ ] Add floating-point marshaling and decoding once the required `strconv`
  float surface is available.
- [x] Implement `Encoder`, `Decoder`, `Encode`, and incremental `Decode`,
  including multiple values in a stream for the supported value set.
- [ ] Implement custom `Unmarshaler` dispatch throughout nested typed values,
  not only at the top-level destination.
- [ ] Add unknown-field controls, number preservation where required, and
  Go-compatible error types and offsets.
- [x] Provide compiler-generated runtime type metadata for struct field names,
  offsets, anonymous status, and tags, exposed through the initial `reflect`
  package surface used by `encoding/json`.
- [ ] Add staragent protocol fixture tests and compare canonical output with Go.
- [ ] Run the JSON regression through the self-hosted compiler after stage0
  passes.

### Remaining inventory

All other packages and hosted-runtime facilities below remain unchecked and are
not complete unless a later checklist entry explicitly says otherwise.

## Packages absent from the bundled standard library

The following production imports have no corresponding package in Renvo's
`std/` tree:

| Package | Staragent usage | Priority |
|---|---|---|
| `bufio` | terminal rune input, scanners, SSE parsing, buffered files | Required |
| `context` | cancellation across agent, HTTP, evaluation, and processes | Required |
| `crypto/sha256` | workspace and trust-document digests | Required |
| `encoding/base64` | JWT payloads and binary workspace data | Required |
| `encoding/hex` | printable SHA-256 values | Required |
| `encoding/json` | API protocol, auth, settings, sessions, trust, JSON UI | Required |
| `flag` | command-line parsing | Required |
| `math` | floating-point equality in the embedded Starlark implementation | Required |
| `net/http` | OpenAI and Codex HTTP clients | Required and substantial |
| `os/exec` | trusted host commands exported from `AGENTS.star` | Required for full functionality |
| `os/signal` | interrupt cancellation | Required for normal CLI behavior |
| `path/filepath` | native paths and directory traversal | Required |
| `regexp` | bounded workspace regex search | Required |
| `syscall` | Unix terminal attributes/ioctl | Required by current platform files |
| `time` | expiry, durations, timestamps, deadlines, status timing | Required |
| `unicode` | rune classification in editing and lexing | Required |

Tests additionally import `net/http/httptest`, which is absent. It is not
needed to build the production command.

`unicode/utf8` is **present** at `std/unicode/utf8`; its API still needs to be
checked against staragent's use of validation, rune decoding, and safe
truncation. Likewise, `sync` exists even though older summary prose may omit it.

## Existing packages requiring API and semantic audit

### `bytes`

Required surface includes `Buffer`, its write/read/string methods,
`NewReader`, and request-body-compatible reader behavior. Buffer and reader
semantics must satisfy the `io` interfaces used by `net/http` and staragent.

### `errors`

Required surface includes `New`, `Is`, and `As`, including traversal through
wrapped errors. Error identity matters for `os.ErrNotExist`, `io.EOF`, and
staragent's own sentinels.

### `fmt`

Required surface includes `Errorf` with `%w`, `Sprintf`, `Fprint`, `Fprintln`,
and `Fprintf`. Formatting must cover strings, quoted values, integers, floats,
types, Go-syntax diagnostics, pointers, errors, widths, and precision. A
formatter that merely accepts calls but mishandles `%w`, `%T`, `%#v`, or float
precision is insufficient.

### `io`

Required surface includes:

- `Reader`, `Writer`, and `ReadCloser`;
- canonical `EOF` and `Discard`;
- `ReadAll`, `LimitReader`, `Copy`, and `NopCloser`.

The canonical identity of `io.EOF` must be retained across `bufio`, `bytes`,
files, and HTTP bodies.

### `os`

Audit at least:

- `Args`, `Stdin`, `Stdout`, and `Stderr`;
- `Getenv`, `UserHomeDir`, and `UserConfigDir`;
- `Open`, `OpenFile`, `CreateTemp`, `ReadFile`, and `WriteFile`;
- `Stat`, `Lstat`, `ReadDir`, `MkdirAll`, `Remove`, and `Rename`;
- `ErrNotExist` and `FileMode`;
- file `Read`, `Write`, `Close`, `Sync`, `Chmod`, `Stat`, `Fd`, and `Name`.

Atomic replacement and restrictive file modes are security requirements for
credential and trust files, not optional compatibility details. Directory
walking must not silently follow symlinks where staragent explicitly avoids it.

### `path`

Staragent uses slash-separated path cleaning and validation. Verify `Clean`,
`Join`, `Base`, `Dir`, and absolute/path traversal behavior against Go.

### `sort`

The embedded Starlark runtime uses callback-based stable sorting. Verify
`Slice`, `SliceStable`, `Strings`, and deterministic behavior where called.

### `strconv`

Audit integer parsing/formatting, float formatting, quoting, and unquoting APIs.
The Starlark lexer and renderer are sensitive to exact numeric and quoted-string
behavior.

### `strings`

The required surface is broad: `Builder`, `Reader`, joining, splitting,
`Fields`, prefix/suffix tests, substring search, trimming, replacement,
repetition, and case conversion. `strings.Reader` must satisfy the interfaces
needed by request construction and buffered input.

### `sync`

Staragent primarily needs zero-value `Mutex` behavior. Correct lock/unlock
semantics are required around credentials, sessions, persistent evaluator
state, and UI state. Renvo's serialized scheduler does not make these locks
unnecessary because callbacks and future concurrency can interleave state.

### `unicode/utf8`

Verify `Valid`, `ValidString`, rune decoding, rune length, and replacement-rune
semantics needed by bounded output and terminal editing.

### `unsafe`

The package exists and Renvo declares unsafe intrinsics frontend work. The
current terminal implementation nevertheless also needs target-correct ioctl
structures and constants from `syscall`.

## Required new package surfaces

### `context`

At minimum implement Go-compatible `Context`, `CancelFunc`, `Background`,
`TODO`, `WithCancel`, and the deadline/cancellation behavior used by HTTP and
process execution. Merely providing types is not enough: blocked streaming I/O
and child processes must observe cancellation.

### `encoding/json`

Staragent requires considerably more than simple struct encoding:

- `Marshal`, `MarshalIndent`, and `Unmarshal`;
- `NewEncoder`, `NewDecoder`, `Encoder.Encode`, and `Decoder.Decode`;
- `RawMessage`;
- struct field tags, omitted fields, pointers, interfaces, maps, slices, and
  anonymous/nested structs;
- custom `MarshalJSON`/`UnmarshalJSON` methods where implemented;
- numeric, boolean, null, escaping, and unknown-field behavior compatible with
  the protocol types.

This package exercises reflection-like type metadata. It should receive focused
compatibility tests using staragent's request and streamed response structures.

### `bufio`

Implement `Reader`, `Writer`, and `Scanner`, including rune reads, line/string
reads, scanner split behavior, `Buffer`, `Text`, `Bytes`, `Err`, and token-size
handling. Staragent raises scanner limits for machine input and SSE data, so the
small default token limit must be configurable.

### Hash and text encodings

Implement Go-compatible SHA-256, standard/raw URL base64 encodings, and hex
encoding. JWT uses unpadded URL-safe base64. Digest output must be stable across
targets.

### `flag`

Support `FlagSet`/command-line parsing and the scalar option types used by
`main.go`, with Go-compatible errors and usage behavior.

### `path/filepath`

Implement native `Clean`, `Join`, `Dir`, `Base`, `Abs`, `Rel`, `IsAbs`,
`ToSlash`, `FromSlash`, `EvalSymlinks` where called, and filesystem walking.
Behavior must be target-sensitive on Windows rather than treating every target
as Unix.

### `regexp`

Staragent exposes user-supplied regex search. Implement compilation, matching,
and errors with bounded execution. Catastrophic patterns must not bypass the
application's file/byte/result limits.

### `time`

Required concepts include `Time`, `Duration`, `Now`, `Since`, `Unix`, `Date`,
`Parse`, comparisons, arithmetic, UTC conversion, and RFC3339/RFC3339Nano
formatting. Timers/deadlines used by context and process execution need an
actual target clock and wakeup mechanism.

### `unicode`

Provide the rune classifications used by word editing and source lexing,
including letter, digit, and space checks with Unicode rather than ASCII-only
behavior.

## Hosted runtime facilities

### HTTPS networking

A functioning `net/http` implementation requires underlying facilities that
are not represented by a package stub:

1. TCP sockets and streaming reads/writes;
2. DNS resolution;
3. TLS 1.2/1.3 client support;
4. system or bundled certificate roots and hostname validation;
5. HTTP request/response parsing, headers, status codes, and bodies;
6. request cloning/replay and reusable body support;
7. cancellation of blocked DNS, connect, TLS, read, and write operations;
8. incremental response reads for server-sent events.

Staragent needs `Client`, `DefaultClient`, `Request`, `Response`, `Header`,
`NewRequestWithContext`, methods/status constants, `Do`, body close semantics,
and enough transport behavior for long-lived streamed responses. Redirect,
proxy, keep-alive, and timeout policy should be explicitly documented if they
differ from Go.

`net/http/httptest` can follow later for tests, backed by a local listener and a
minimal server, or tests can initially receive Renvo-specific transport
fixtures.

### Process execution

Current code imports `os/exec` and needs command construction, a working
directory, environment, standard streams, exit status, and cancellation. Renvo's
nonstandard `process` package may be a suitable backend for a Go-compatible
adapter. This feature should be supported only on hosted targets where process
creation exists; WASI, browser, VM, and embedded targets should return a clear
unsupported error or compile a deliberately reduced feature set.

### Signals and terminals

The existing Linux/macOS files import `syscall` and perform raw-terminal ioctls.
Two viable approaches are:

- implement the needed Go-compatible `syscall` constants, `Termios`, and
  `Syscall` behavior per target; or
- add `renvo`-tagged staragent adapters backed by a smaller Renvo terminal API.

A target adapter is preferable to a broad legacy `syscall` implementation.
`os/signal.NotifyContext` or equivalent cancellation is also needed for Ctrl-C.
An initial machine-JSON-only build could omit raw line editing, but that is a
reduced product rather than full compatibility.

## Language and compiler feature assessment

No known source construct in staragent is forbidden by Renvo's documented full
frontend. The project does use features outside the backend's direct source
subset, including:

- maps and interfaces;
- closures and function values;
- methods and dynamic dispatch;
- `defer`, `panic`, and `recover` behavior;
- named and multiple returns;
- type assertions and type switches;
- variadic calls;
- struct tags;
- range over maps, slices, strings, and integers;
- platform build constraints;
- unsafe pointer conversion.

These are expected to be lowered by the full frontend. `backend/docs/SUBSET.md`
describes direct backend source tests and must not be mistaken for the frontend
acceptance policy.

Areas worth adding as focused frontend regressions are:

- `for range n` and use of predeclared `min`, `max`, and `clear`;
- large nested composite protocol structs with JSON tags;
- maps whose values are interfaces, slices, or `json.RawMessage`;
- interface-wrapped errors with `%w`, `errors.Is`, and `errors.As`;
- closures passed to stable sorting;
- deferred close/unlock operations on early return;
- build selection for `linux`, `darwin`, `windows`, and `renvo` tags;
- unsafe terminal structures on each native architecture.

## Runtime behavior risks after compilation

Renvo uses bounded arenas rather than Go's tracing garbage collector.
Staragent is long-running and repeatedly allocates response trees, JSON buffers,
conversation history, scan results, Starlark values, and command output. Even a
correctly compiled program may exhaust the default hosted arena over many turns.
Validation must include a long multi-turn session under the intended arena
limit, not just startup and one request. Conversation compaction, bounded output,
and buffer reuse help, but an arena-size policy or lifetime-oriented changes may
still be necessary.

Renvo concurrency is cooperative and serialized under the reference handler.
The current code mostly relies on mutexes and synchronous calls, so parallelism
is not required. Future goroutines would require installing a Renvo runtime
handler before the first concurrency operation.

Target expectations must be explicit:

- Full interactive functionality is initially realistic on hosted native
  targets.
- WASI lacks ordinary child processes and may lack unrestricted sockets.
- Browser output cannot expose host process execution or raw terminals.
- VM32 and embedded targets need host-provided network/process capabilities or
  a reduced build.

## Suggested implementation order

1. Add low-level, broadly reusable packages: `unicode`, `time`, hashing,
   base64, hex, `path/filepath`, and `bufio`.
2. Complete the existing `io`, `os`, `bytes`, `errors`, `fmt`, `strings`,
   `strconv`, `sort`, and `unicode/utf8` surfaces with compatibility tests.
3. Implement `context`, `flag`, `regexp`, and full `encoding/json` support.
4. Add Renvo-specific terminal and interrupt adapters so a native CLI can run
   without requiring all of legacy `syscall`.
5. Build the socket/DNS/TLS/HTTP stack and verify streamed cancellation.
6. Adapt Renvo's process facilities behind `os/exec`, with clear unsupported
   behavior on non-hosted targets.
7. Compile staragent with tests excluded, then run a local mock transport test.
8. Add `net/http/httptest` and run the staragent test suite under Renvo.
9. Run a real authenticated streaming request and a long arena-soak session.
10. Validate each claimed native target separately, especially terminal ABI,
    path semantics, TLS roots, signal handling, and process creation.

## Completion criteria

The port is not complete merely when the package type-checks. A full native
port should demonstrate:

- clean Renvo compilation of the production package;
- settings, auth, trust, and session files round-trip correctly;
- an HTTPS request validates certificates and streams SSE incrementally;
- Ctrl-C cancels an active model stream;
- raw terminal mode is restored after normal exit, error, and interrupt;
- trusted host commands return bounded stdout/stderr and honor cancellation;
- workspace path and symlink boundaries remain enforced;
- SHA-256 and encoding results match Go;
- JSON request/response fixtures match Go byte-for-byte where canonical output
  is expected and semantically elsewhere;
- a representative test partition passes under `renvo test`;
- a sustained multi-turn run stays within the selected arena policy.
