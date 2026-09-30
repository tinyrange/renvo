# Bundled C runtime

This is a small target-side libc, not a host-libc forwarding layer or a complete
C/POSIX implementation. The compiler preprocesses the selected libc sources
with the user translation unit, then lowers them to target code. Headers can
be bundled with `renvo_bundle`; explicit include paths take precedence and
`-nostdinc` disables the bundled fallback.

## Added hosted facilities

- Unbuffered file streams: `fopen`, `fclose`, `fflush`, `fileno`, character,
  line and block I/O, one-byte pushback, EOF/error indicators and `clearerr`.
  Modes support read/write/append, update, binary and exclusive creation.
  A compact reusable pool provides `FOPEN_MAX == 16`, including the three
  standard streams, without allocating a new stream on every reopen. The
  bounded storage matters for single-segment targets such as DOS COM.
- Single-threaded `errno`, common error constants and `strerror`. Negative
  target descriptor results become stream errors; formatted output reports
  write failures. `__fpending` is zero because writes are unbuffered.
- Process environment lookup through `getenv`; up to 32 `atexit` callbacks,
  invoked in reverse order by `exit` and normal return from hosted `main`.
  `_Exit` and `_exit` bypass callbacks.
- C/POSIX and C.UTF-8/C.utf8 locale selection, including environment precedence
  and per-category selection. Unsupported locale names fail without changing
  the current categories. This is not a locale database or gettext support.
- Restartable `mbrtowc`, `mbrlen`, `wcrtomb`, `mbsrtowcs`, `wcsrtombs` and
  `mbsinit`, with strict UTF-8 scalar validation, partial input and bounded
  output. The C encoding accepts ASCII only.
- Basic wide strings and wide output. Wide formatting currently supports
  strings, characters, percent and a subset of integer conversions. Unsupported
  conversions fail; width, precision and floating-point wide formatting are
  not implemented. Hosted ASCII wide literals are lowered to target storage.

The descriptor bridge uses Renvo target operations, not compiler-host file
access. On Linux/amd64, stream creation uses the target open syscall with mode
0666 so the process umask applies atomically and existing modes are retained.
Other target bridges still use the portable two-argument open operation; the
creation-mode fix has not been established for those targets.
The tested file/locale/exit runtime is Linux/amd64; the DOS COM smoke
continues to enforce its original image-size limit. Other targets require
their own runtime acceptance checks, including native error-number mappings.

## Target headers and build compatibility

The Linux/amd64 sysroot now supplies `sys/types.h`, `sys/stat.h`, `time.h`,
`fcntl.h`, `sys/resource.h` and `stdnoreturn.h`. The Unix type/stat headers explicitly reject
unsupported target ABIs instead of borrowing host layouts. The stat record is
144 bytes, with checked field offsets. Time, stat and open-family declarations
are **not** evidence of runtime implementations; those APIs remain incomplete.
Linux/amd64 `fcntl` implements descriptor duplication, descriptor flags and
file-status flags through the target syscall. The virtual Linux provider supports
shared append-mode changes, preserves per-descriptor close-on-exec flags, and
explicitly rejects unsupported controls rather than claiming success.

The `cc` object-link path resolves actual bundled objects from ELF undefined
symbols: allocation, strings, streams, locale/wide characters, ctype, file
control, resource limits and their syscall bridges. `-nostdlib` and standalone
`ld` do not add these objects. Providers are selected only for unresolved
symbols, not when a user definition already satisfies that reference; ordinary
strong-symbol collision rules still apply to other exports in a selected object.
Headerless function-presence probes link without merging incompatible prototypes.
Hosted object startup initializes argv/environment and GNU program names, then
uses `exit(main(...))` to run callbacks on normal return. Native and virtual
acceptance execute the same final ELF.

Linux/amd64 `getrlimit` forwards to syscall 97 with the ABI's two 64-bit fields;
errors set errno. The trex virtual kernel implements RLIMIT_NOFILE with its
actual fixed descriptor capacity and rejects unsupported resources explicitly.
`strstr` and bounded `strnlen` are also provided.

Linux/amd64 hosted additions include GNU program-name globals, `strdup`,
`strndup`, and descriptor `dup2`; `close` uses the existing descriptor bridge.
Program names initialize from argv even when it is empty. `MB_CUR_MAX` follows
LC_CTYPE (one for C, four for UTF-8), rather than advertising a fixed locale.

Narrow buffer formatting (`sprintf`, `snprintf`, `vsprintf`, `vsnprintf`) shares
the existing stream formatter. Truncation preserves the would-have-written
count and terminates nonempty buffers; size zero accepts a null destination.
This does not extend the existing limited conversion/width/precision support
to the full printf specification.

The compiler now preprocesses system headers in ordinary `cc -c` mode rather
than building declarations from raw header text. Multiline preprocessing
comments, concatenated static-assert messages, C11 `_Noreturn`, and parenthesized
string-pointer casts have regression coverage. `sizeof` no longer accepts
nonexistent aggregate fields or parenthesized typedef names as expressions,
which previously corrupted configure feature detection.

Linux-only hosted helpers are not pulled into the DOS runtime: the original
COM image-size limit remains enforced by its existing smoke test.

## Known limits

There is no seeking, configurable buffering, stream orientation, wide input,
thread-safe errno/locale state, or complete POSIX API. `unistd.h` provides
standard descriptor constants, `_exit`, `close`, and Linux/amd64 `dup2`.
Narrow formatted output retains its limited format implementation. Allocation is still
bump-based; `free` does not reclaim storage. The parent virtual environment now completes the original GNU Hello 2.12.1
configure, build and all seven upstream tests (zero skips) in MemoryFS. It
validates the exact generated ELF natively and virtually across normal output,
help/version, argument errors, long greetings and write failures. This proves
that specific Linux/amd64 C-locale workload, not complete Gnulib or POSIX
compatibility. Source/config/object/archive stages never use host tools or
host intermediary files.

## Focused validation

`libc/tests/streams.c` checks stream modes, contents, append/update behavior,
EOF/pushback/errors, partial block reads, zero-length I/O, pool exhaustion and
slot reuse. `libc/tests/wide_exit.c` checks environment/locale, invalid and
partial UTF-8, bounded conversions, wide output and callback ordering.

`backend/tests/c_header_semantics.c` checks stat ABI offsets, formatting bounds
and varargs, string duplication, locale limits, GNU process names, and descriptor
duplication/closure. The native and virtual tests execute that same source.
The virtual suite also checks empty argv and descriptor alias lifetimes.
`libc/tests/fcntl.c` checks shared append behavior and descriptor flags;
`fcntl_object.c` exercises the separate-object runtime bridge without headers.
Both execute natively and in the virtual Linux environment.
`libc/tests/object_runtime.c` covers separately linked startup, argc/argv,
environment values, GNU program names, getrlimit/EFAULT, strstr/strnlen and
normal-return exit callbacks, with identical native/virtual final executables.
The C11 unary-sizeof and inline-linkage regressions live in `backend/tests/`
with native frontend acceptance in `frontend_tests/`.

Run native acceptance in this checkout:

```sh
GOWORK=off go test -tags renvo_bundle ./frontend_tests -run '^TestFrontendC(HeaderSemantics|FileStreams|WideAndExit)$'
GOWORK=off go test ./internal/backendjit -run '^TestMSDOSCOMExample$'
```

The trex parent executes the same C fixtures entirely in its virtual filesystem
and Linux emulator, and separately checks an output failure during exit:

```sh
go test -race -tags renvo_bundle ./emulator/buildenv ./emulator/linux ./emulator/shell
```

These are focused checks, not a replacement for the repository preflight.
