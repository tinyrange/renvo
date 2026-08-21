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

The initial `context` package now clears this loader blocker. Recompiling the
external staragent source is the next way to identify the following missing
package in import order; the remaining entries below still come from the complete
import/tree audit.

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
- [x] `crypto/sha256`: one-shot `Sum256`, `Size`, and `BlockSize`.
  - [x] Host-Go tests cover empty, short, and multi-block standard vectors.
  - [x] `frontend_tests/regressions/std_sha256` verifies the same vectors in a
    Renvo-compiled executable.
  - [ ] Add the streaming `New`/`hash.Hash` surface if a caller requires it.
- [x] `encoding/base64`: standard and URL alphabets, padded and raw encodings,
  byte encode/decode, and string helpers.
  - [x] Host-Go package tests cover padded standard and raw URL round trips.
  - [x] The focused Renvo regression covers the unpadded URL encoding path.
  - [x] Compiled regression rejects malformed characters, impossible one-byte input, and padding in raw URL input.
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
  - [x] Host-Go and compiled regressions cover empty/nested values and non-empty prefix behavior.
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
- [x] Run the JSON regression through a native `darwin/arm64` self-hosted
  compiler. Explicit-file and package-path invocation both compile and their
  executables print `PASS`; nested module roots are resolved from the requested
  input rather than the caller's working directory.

### Initial `bufio.Scanner` work

- [x] Added host-Go implementations of `Scanner`, `SplitFunc`, `NewScanner`,
  `Scan`, `Bytes`, `Text`, `Err`, `Split`, and `Buffer`.
- [x] Added `ScanLines`, `ScanWords`, `ScanBytes`, and `ScanRunes`, with host
  tests for CRLF/final lines, words, UTF-8 runes, the default token limit, and a
  caller-supplied larger buffer.
- [x] Execute Scanner through Renvo, including imported `SplitFunc` callbacks and
  function-valued scanner fields.
- [x] Added `Reader`, including buffered reads, byte/rune reads and unread,
  peeking, discarding, line/slice/byte/string reads, reset, size, and buffered
  byte reporting.
  - [x] Host-Go tests cover line boundaries, CRLF, EOF, buffer-full assembly,
    peeking/discarding, and UTF-8 unread behavior.
  - [x] A focused Renvo regression covers peeking, discarding, and line reads.
  - [x] Renvo execution covers repeated three-result `ReadLine` calls and the
    rune path without result corruption.
- [x] Added `Writer`, including sized construction, buffered byte/string/rune
  writes, direct large writes, flush, reset, available/buffered reporting,
  `AvailableBuffer`, and `ReadFrom`.
  - [x] Host-Go tests cover buffering and flush, UTF-8 runes, reset, large
    direct writes, `ReadFrom`, and sticky short-write errors.
  - [x] `frontend_tests/regressions/std_bufio_writer` executes buffered writes,
    UTF-8 output, flush, reset, and `ReadFrom` through Renvo.

### Initial `path/filepath` work

- [x] Added Unix lexical path operations: `Clean`, `Join`, `Split`, `Dir`,
  `Base`, `Ext`, `Abs`, `Rel`, `IsAbs`, `VolumeName`, `ToSlash`, `FromSlash`,
  separators, and `IsPathSeparator`.
- [x] Added pattern matching with `*`, `?`, character classes, ranges,
  negation, escaping, and `ErrBadPattern`.
- [x] Added filesystem-backed `Glob` using sorted `os.ReadDir` results and
  Go-style suppression of filesystem I/O errors.
- [x] Host-Go tests cover lexical cleaning and traversal, relative paths,
  matching, malformed patterns, and deterministic globbing.
- [ ] Add Windows volume, UNC, separator, cleaning, and matching semantics;
  the current implementation is Unix-only.
- [ ] Add `EvalSymlinks`, `Walk`, and `WalkDir`. These cannot be implemented
  correctly over the current Renvo `os` surface: it has no `Stat`, `Lstat`,
  `Readlink`, `FileInfo`, or `io/fs` API, and the allowed syscall set has no
  metadata/readlink operation. They must not be replaced with lexical cleaning
  because staragent relies on symlink boundaries for security.
- [ ] Execute the focused path regression through Renvo. Compilation produced
  a non-native executable in the current test configuration (`exec format
  error`), so no semantic compiled result was claimed.

### Initial `unicode` work

- [x] Added generated Unicode-property tables and `IsLetter`, `IsDigit`,
  `IsNumber`, `IsSpace`, `IsUpper`, `IsLower`, `ToUpper`, and `ToLower`.
- [x] Tables and simple case mappings are generated from host Go's Unicode
  database and checked in; they cover the full rune range rather than selected
  scripts.
- [x] Host-Go tests cover Latin, Greek, Cyrillic, Arabic, Devanagari, fullwidth
  digits, CJK, and non-ASCII whitespace and case mapping.
- [x] `frontend_tests/regressions/std_unicode` passes through Renvo.

### Initial `flag` work

- [x] Added `FlagSet`, `Flag`, `Value`, `Getter`, error-handling modes,
  scalar string/bool/int/int64/uint registration, parsing, lookup, setting,
  positional arguments, `--`, visitation, and package-level `CommandLine`
  helpers.
- [x] Parsing supports `-name value`, `-name=value`, `--name=value`, and
  implicit true values for boolean flags.
- [x] Host-Go tests cover scalar parsing, errors, positional arguments,
  terminators, counts, and sorted visitation.
- [x] `frontend_tests/regressions/std_flag` passes through Renvo.
- [x] Added float and duration values, `SetOutput`, `Output`,
  `ErrorHandling`, `PrintDefaults`, and `UnquoteUsage`, plus standard
  stdin/stdout/stderr handles needed for default output.
  - [x] Host-Go tests cover decimal/exponent floats, compound/fractional
    durations, usage metavariables, and captured defaults output.
  - [x] Renvo execution covers float/duration parsing and the Go-style default
    `FlagSet.Usage` closure.

### Foundational package audit progress

- [x] `io`: `Reader`, `Writer`, `Closer`, `ReadCloser`, canonical `EOF`,
  `ErrShortWrite`, `Discard`, `ReadAll`, `LimitReader`, `Copy`, `NopCloser`, and
  `WriteString`, with host-Go compatibility tests.
- [x] `bytes`: byte helpers, `Buffer`, and request-body-compatible `NewReader`
  with `Read`, `Len`, `Size`, and `Reset`, with host-Go compatibility tests.
- [x] `strings`: `Builder`, request-body-compatible `Reader`, joining, splitting,
  fields, prefix/suffix and substring operations, trimming, replacement,
  repetition, and Unicode case conversion, with host-Go compatibility tests.
- [x] `unicode/utf8`: validation for strings and byte slices, rune decoding,
  rune counts and lengths, encoding, and replacement-rune semantics, with
  host-Go compatibility tests.
- [x] `time`: duration parsing/formatting plus UTC/fixed-offset calendar values,
  Unix conversion, arithmetic, comparisons, and RFC3339/RFC3339Nano parsing and
  formatting, with host-Go tests and a Renvo-compiled regression.
  - [x] `Now` reads native realtime and monotonic clocks on Linux and
    darwin/arm64; `Since` uses monotonic elapsed time when both values carry it.
  - [ ] Timers, sleeps, deadline wakeups, named locations, and general Go layouts
    still require additional runtime and package support.
- [x] `errors`: identity-preserving `New`, single and multi-error traversal for
  `Is`, custom `Is`/`As`, `Unwrap`, `Join`, and direct assignment to `*error`,
  with host-Go and Renvo-compiled coverage.
  - [ ] General assignable typed-target handling in `As` still needs reflective
    interface assignment; custom `As` methods and `*error` targets work.
- [x] `strconv`: booleans, integer formatting/parsing across bases, bit-size
  overflow saturation, `NumError`, decimal/exponent float parsing, initial
  `f`/`e`/`g` formatting, and basic quoting/unquoting, with host-Go and compiled
  coverage.
  - [ ] Hex floats, `NaN`/`Inf`, exact shortest-round-trip float formatting,
    and the complete Go quote/unquote escape surface remain.
- [ ] Complete the remaining audited surfaces in `fmt`, `os`, and `sort`.

### Initial `context` work

- [x] Added `Context`, `CancelFunc`, `CancelCauseFunc`, `Background`, `TODO`,
  `WithCancel`, `WithCancelCause`, `Cause`, and `WithValue`.
- [x] Host-Go tests cover empty contexts, nested values, cancellation trees,
  cause propagation, idempotent cancellation, and child/parent independence.
- [x] Renvo execution covers creating and invoking a returned `CancelFunc` and
  observing `Canceled` through `Err` with the serial runtime installed.
- [ ] Receiving from `Context.Done()` through an interface currently fails in
  frontend linking; the focused failure is recorded in `COMPILER_BUGS.md`.
- [ ] Add `WithDeadline`, `WithTimeout`, timer wakeups, and external-parent
  cancellation observation.

### Remaining inventory

All other packages and hosted-runtime facilities below remain unchecked and are
not complete unless a later checklist entry explicitly says otherwise.

## Packages absent from the bundled standard library

The following production imports have no corresponding package in Renvo's
`std/` tree:

| Package | Staragent usage | Priority |
|---|---|---|
| `math` | floating-point equality in the embedded Starlark implementation | Required |
| `net/http` | OpenAI and Codex HTTP clients | Required and substantial |
| `os/exec` | trusted host commands exported from `AGENTS.star` | Required for full functionality |
| `os/signal` | interrupt cancellation | Required for normal CLI behavior |
| `regexp` | bounded workspace regex search | Required |
| `syscall` | Unix terminal attributes/ioctl | Required by current platform files |

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

The implemented surface includes identity-preserving `New`, `Unwrap`, `Join`,
and single/multi-error traversal through `Is` and `As`, including custom
`Is(error) bool` and `As(any) bool` methods. `As` directly assigns `*error`
targets; general typed-target assignment still needs reflective interface
assignment. Error identity is retained for sentinels.

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

The implemented surface covers boolean conversion, integer parsing/formatting
with base prefixes and bit-size overflow, `NumError`, decimal/exponent float
parsing, initial `f`/`e`/`g` formatting, append helpers, and basic string quoting
and unquoting. Hex floats, special values, exact shortest-round-trip formatting,
and the complete Go escape grammar remain; the Starlark lexer and renderer need
compatibility fixtures before this package can be considered exact.

### `strings`

The audited staragent surface is implemented: `Builder`, `Reader`, joining,
splitting, `Fields`, prefix/suffix tests, substring search, trimming,
replacement, repetition, and Unicode case conversion. `strings.Reader`
satisfies `io.Reader` for request construction and buffered input.

### `sync`

Staragent primarily needs zero-value `Mutex` behavior. Correct lock/unlock
semantics are required around credentials, sessions, persistent evaluator
state, and UI state. Renvo's serialized scheduler does not make these locks
unnecessary because callbacks and future concurrency can interleave state.

### `unicode/utf8`

The audited surface provides `Valid`, `ValidString`, byte and string rune
decoding, rune counts and lengths, encoding, and replacement-rune semantics
needed by bounded output and terminal editing.

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

One-shot SHA-256 plus standard/raw URL base64 and hex encoding are implemented.
JWT uses unpadded URL-safe base64. Add streaming hash APIs if required, and keep
digest output stable across targets.

### `flag`

`FlagSet` parsing now includes integer, boolean, string, float, and duration
options plus captured defaults output. Renvo still needs backend support for the
default usage closure and compiled float/duration paths.

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

The initial surface implements `Time`, `Duration`, `Unix`, `Date`, `Parse`,
comparisons, arithmetic, UTC/fixed-offset conversion, and RFC3339/RFC3339Nano
formatting. `Now` uses native realtime plus monotonic clocks on Linux and
Darwin/arm64, and `Since` preserves monotonic elapsed-time semantics. Timers,
sleeps, deadline wakeups, named locations, and arbitrary layouts remain.

### `unicode`

Generated full-range Unicode letter, digit, number, space, and case APIs are
implemented and exercised through Renvo.

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
