# PDP-11 Unix V7 browser-backend demo

This example is a small but complete custom backend demonstration. The browser
opens [`pdp11_v7.rtg`](pdp11_v7.rtg), discovers `unixv7/pdp11`, JIT-compiles a
VM32 backend compiler, caches it by the RTG target identity, and uses that
compiler to turn [`main.go`](main.go) into a PDP-11 executable.

The output is an original Unix V7 `0407` `a.out`, not an emulator container or
a prebuilt byte sequence. The RTG definition supplies PDP-11 instruction
encoding, the internal calling convention, V7 `write` and `exit` traps, packed
data handling, relocations, and the 16-byte executable header. Its intentionally
small scope is the Hello World demo and similarly simple 16-bit programs.

From the repository root, the equivalent command-line build is:

```sh
go run ./cmd/renvo \
  -backend examples/pdp11v7/pdp11_v7.rtg \
  -t unixv7/pdp11 -s -o sandbox/hello.pdp11 \
  examples/pdp11v7
```

With the companion V7 workspace installed at `~/dev/projects/retro/pdp11-unix`,
copy and run it with:

```sh
cd ~/dev/projects/retro/pdp11-unix
./put-files.sh /usr/josh ~/dev/projects/renvo/sandbox/hello.pdp11
./run.sh
```

At the V7 prompt:

```text
# cd /usr/josh
# ./hello.pdp11
Hello from Renvo on PDP-11 Unix V7!
```

Unix V7 limits filenames to 14 bytes, so keep the transferred output name
short.
