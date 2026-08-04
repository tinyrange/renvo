//go:build !renvo && darwin && arm64

#include "textflag.h"

TEXT darwinMmapTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_mmap(SB)
GLOBL ·darwinMmapAddr(SB), RODATA, $8
DATA ·darwinMmapAddr(SB)/8, $darwinMmapTrampoline<>(SB)

TEXT darwinMprotectTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_mprotect(SB)
GLOBL ·darwinMprotectAddr(SB), RODATA, $8
DATA ·darwinMprotectAddr(SB)/8, $darwinMprotectTrampoline<>(SB)

TEXT darwinMunmapTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_munmap(SB)
GLOBL ·darwinMunmapAddr(SB), RODATA, $8
DATA ·darwinMunmapAddr(SB)/8, $darwinMunmapTrampoline<>(SB)

TEXT darwinDlopenTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_dlopen(SB)
GLOBL ·darwinDlopenAddr(SB), RODATA, $8
DATA ·darwinDlopenAddr(SB)/8, $darwinDlopenTrampoline<>(SB)

TEXT darwinDlsymTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_dlsym(SB)
GLOBL ·darwinDlsymAddr(SB), RODATA, $8
DATA ·darwinDlsymAddr(SB)/8, $darwinDlsymTrampoline<>(SB)

TEXT darwinDlcloseTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_dlclose(SB)
GLOBL ·darwinDlcloseAddr(SB), RODATA, $8
DATA ·darwinDlcloseAddr(SB)/8, $darwinDlcloseTrampoline<>(SB)

TEXT darwinInvalidateInstructionCacheTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_sys_icache_invalidate(SB)
GLOBL ·darwinInvalidateInstructionCacheAddr(SB), RODATA, $8
DATA ·darwinInvalidateInstructionCacheAddr(SB)/8, $darwinInvalidateInstructionCacheTrampoline<>(SB)

TEXT darwinJITWriteProtectTrampoline<>(SB),NOSPLIT,$0-0
	JMP libc_pthread_jit_write_protect_np(SB)
GLOBL ·darwinJITWriteProtectAddr(SB), RODATA, $8
DATA ·darwinJITWriteProtectAddr(SB)/8, $darwinJITWriteProtectTrampoline<>(SB)
