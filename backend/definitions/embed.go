package definitions

import "embed"

// WindowsKernelAMD64 contains the backend sources needed to compile an AMD64
// Windows kernel driver in a closed virtual filesystem.
//
//go:embed drivers/windows_kernel_amd64.rtg x86_64.rtg elf_amd64_primitives.rtg
var WindowsKernelAMD64 embed.FS
