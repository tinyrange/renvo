# Compiler bugs

## `uint64` division in a conversion-return expression

Found while compiling `examples/uefi-linux-boot` for `uefi/amd64` through the
external backend definition. This valid helper fails during prepared-backend
emission:

```go
func pagesFor(bytes uint64) uintptr {
	return uintptr((bytes + 4095) / 4096)
}
```

The compiler reports:

```text
failed to emit statement: return uintptr((bytes + 4095) / 4096)
failed function: pagesFor
prepared backend failed queued functions
```

Using a `uintptr` parameter and `(bytes + 4095) >> 12` works around the issue.
The original form should either compile or produce a precise diagnostic for an
intentionally unsupported operation.

## Escaping control storage around firmware output pointers

Found in the UEFI `GetMemoryMap` wrapper. A local struct/array used for four
firmware output values was followed by a dynamic byte buffer whose backing
address overlapped the apparent control storage. OVMF then faulted while
writing the memory map. Keeping the 32-byte control block and map bytes in one
explicit byte allocation, and addressing each field by byte offset, works
around the issue. This needs a reduced frontend/backend reproducer before the
fault can be attributed more narrowly to escape analysis, aggregate sizing, or
unsafe address lowering.
