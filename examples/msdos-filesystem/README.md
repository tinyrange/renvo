# MS-DOS filesystem demo

This MZ program uses the compact `device/dos` handle API to create, write,
open, read, and close a file, then demonstrates DOS-specific rename,
attributes, directory enumeration, and removal operations without pulling the
generalized `os` implementation into the constrained 16-bit program segment.

```sh
go run ./cmd/renvo -backend backends/msdos.rtg \
  -t msdos/8086-mz -s -o sandbox/files.exe examples/msdos-filesystem
```
