# PDP-11 Unix V7 backend enablement

[`pdp11_v7.rbe`](pdp11_v7.rbe) is a complete Renvo Backend Enablement file.
Its RTG portion defines the PDP-11 instruction encoder, Renvo ABI, original
Unix V7 process entry and syscall conventions, relocations, and `0407` a.out
image. Its `@stdlib` sections add the target-specific `syscall` API and adapt
the portable `os` package for V7 paths, open/creat, and fixed-size directory
records. No frontend source change or separately installed target library is
needed to add `unixv7/pdp11`.

The exposed V7 API includes file and directory operations, process identity and
credentials, time and accounting calls, signals, terminal settings, and the
typed native `stat`, `tms`, `timeb`, `utimbuf`, and `sgttyb` layouts. Calls that
would be unsafe or unusable through a practical Go API, such as raw `exec`,
`mount`, `ptrace`, `phys`, and `lock`, are intentionally not wrapped.

The example also contains `answer_unixv7_pdp11.rtgasm`. It implements a
bodyless Go function with the RBE's normal PDP-11 emitter helpers and exercises
project RTGASM alongside the generated backend.

Build the example from the repository root:

```sh
go run ./cmd/renvo \
  -backend examples/pdp11v7/pdp11_v7.rbe \
  -t unixv7/pdp11 -arena-size 12288 \
  -o sandbox/hello.pdp11 examples/pdp11v7
```

The result is a real PDP-11 V7 executable, not an emulator wrapper or prebuilt
payload. Unix V7 filenames are limited to 14 bytes, so use a short transfer
name.

For a tape-based OpenSIMH transfer, GNU tar's V7 format and 10 KiB tape records
are compatible with the stock V7 `tar` command:

```sh
tar --format=v7 -C sandbox -cf /tmp/renvo.tar hello.pdp11
mktape /tmp/transfer.tap create /tmp/renvo.tar:10240
```

Attach the tape as `tm0`, boot the installed V7 system, and extract with:

```text
# tar xbf 20 /dev/rmt0
# chmod 755 hello.pdp11
# ./hello.pdp11
Hello from Renvo on PDP-11 Unix V7!
```

The backend and its `os`/`syscall` overlays are covered by host-side RBE and
compiler tests. They were also run under OpenSIMH against a genuine V7 install,
including argv/environment decoding, create/write/close, reopen/read, byte
comparison, path access, link/unlink, chmod, and typed stat/fstat calls.
The final guest suite also validates sorted `os.ReadDir` entries and directory
types, `times`, numeric argv values, and a non-empty inherited environment.
