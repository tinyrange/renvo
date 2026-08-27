# UEFI hello

This application prints through the firmware's Simple Text Output protocol,
reads the firmware vendor from `EFI_SYSTEM_TABLE`, and waits for a key through
Simple Text Input.

```sh
go run ./cmd/renvo -backend backends/uefi_amd64.rtg \
  -t uefi/amd64 -s -o sandbox/BOOTX64.EFI examples/uefi-hello
```
