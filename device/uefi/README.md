# UEFI devices

`renvo.dev/device/uefi` is the firmware API for `uefi/amd64` applications. It
provides the image handle and system table, text console access, boot and
runtime service calls, page and pool allocation, memory-map access, protocol
lookup, GOP framebuffer graphics, and Simple File System file access.

The package deliberately follows UEFI's tables and status values closely.
Firmware function pointers cross one small target-specific RTGASM bridge that
implements the Microsoft x64 calling convention; protocol policy remains
ordinary Go. `Call` supports up to five firmware arguments, which covers the
typed services currently exposed by the package.

The target runtime sends ordinary Go `print`, `println`, and `fmt` output to
the UEFI text console. `WriteUTF16` remains available when code needs to pass
an already encoded firmware string directly.

Firmware tables are read by their UEFI byte offsets instead of being cast to
ordinary Go structs. Renvo gives small Go struct fields compact-target-friendly
word slots, while UEFI tables use the packed C ABI; keeping that boundary
explicit prevents an innocent Go layout change from shifting a protocol entry.

Build an application as the removable-media fallback image with:

```sh
go run ./cmd/renvo -backend backends/uefi_amd64.rtg \
  -t uefi/amd64 -s -o sandbox/BOOTX64.EFI examples/uefi-hello
```

See `tools/uefi/test-qemu` for the opt-in OVMF/QEMU smoke test. This is a Tier 3
target: it is tested locally and is not a required CI backend.

The `examples/uefi-linux-boot` project demonstrates a complete native x86-64
Linux boot handoff. Linux-specific image parsing and machine entry code stays
in that example; the general package only supplies the underlying UEFI calls.
