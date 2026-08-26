# DOS devices

`renvo.dev/device/dos` is the hardware- and DOS-specific companion to portable
packages such as `os`. It is selected by the `msdos` and `i8086` target tags and
uses project RTGASM only for the small instruction sequences that cross into
BIOS, DOS, VGA memory, or I/O ports.

The package provides:

- DOS drives, directories, attributes, rename/remove, FindFirst/FindNext,
  conventional-memory allocation, version, console output, and error codes;
- BIOS keyboard, clock, mouse, serial, and printer services;
- VGA mode 13h and 640x480 planar access, palettes, vertical retrace, text
  cursor/teletype operations, and bounded framebuffer transfers;
- PIT and PC-speaker tone control;
- an explicit `Interrupt` register API for services not yet given a typed
  wrapper.

Portable file reads and writes should still prefer `os`. Directory enumeration
uses this package's DOS finder because importing the complete device surface
inside `os` would consume too much of a 16-bit program segment.

The current compiler uses 16-bit near pointers. Both COM and MZ output therefore
keep code, ordinary data, BSS, heap, and stack in one load segment. Explicit
segment operations let VGA and conventional-memory APIs reach other real-mode
segments without pretending that Go pointers are far pointers.
