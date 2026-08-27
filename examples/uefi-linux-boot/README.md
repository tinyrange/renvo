# Linux bootloader

This example is a small UEFI bootloader for 64-bit Linux `bzImage` kernels. It
reads `config.txt` from the EFI System Partition, loads the kernel and
initramfs, builds Linux's x86 boot parameters and physical-memory map, exits
UEFI boot services, then jumps to the kernel's native 64-bit entry point.

The layout and entry contract follow the official
[Linux x86 boot protocol](https://docs.kernel.org/arch/x86/boot.html).

It does not call UEFI `LoadImage` and does not rely on `CONFIG_EFI_STUB`.

Keep these files in the root of the EFI System Partition beside
`EFI/BOOT/BOOTX64.EFI`:

```text
kernel=VMLINUZ
initramfs=INITRAMFS
cmdline=console=ttyS0,115200 rdinit=/init panic=-1
```

The kernel must be a relocatable x86-64 `bzImage` implementing boot protocol
2.12 or newer. Initramfs images are limited to 64 MiB. Run
`tools/uefi/test-linux-qemu` to build a minimal Alpine initramfs and exercise
the complete handoff under OVMF.
