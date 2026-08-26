# DOS devices

`renvo.dev/device/dos` is the compact hardware and operating-system API for DOS
programs. It is selected by the `msdos` and `i8086` target tags and uses project
RTGASM only for the small instruction sequences that cross into BIOS, DOS, VGA
memory, or I/O ports.

The package provides:

- DOS file handles, drives, directories, attributes, rename/remove,
  FindFirst/FindNext, conventional-memory allocation, version, console output,
  and error codes;
- BIOS keyboard, clock, mouse, serial, and printer services;
- VGA mode 13h and 640x480 planar access, palettes, vertical retrace, text
  cursor/teletype operations, and bounded framebuffer transfers;
- PIT and PC-speaker tone control;
- an explicit `Interrupt` register API for services not yet given a typed
  wrapper.

DOS programs should normally use the thin `OpenFile`, `CreateFile`, and finder
APIs rather than generalized filesystem packages. The generalized APIs remain
available, but often consume too much of the single 16-bit program segment once
combined with a useful application.

The current compiler uses 16-bit near pointers. Both COM and MZ output therefore
keep code, ordinary data, BSS, heap, and stack in one load segment. Explicit
segment operations let VGA and conventional-memory APIs reach other real-mode
segments without pretending that Go pointers are far pointers.

The supported backend is [`backends/msdos.rtg`](../../backends/msdos.rtg). Use
target `msdos/8086` for a compact `.COM` image or `msdos/8086-mz` for a
relocatable MZ `.EXE`. The examples under `examples/msdos*` are also available
from the web IDE's IBM PC compatible catalog.
