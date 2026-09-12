//go:build !renvo

#include "textflag.h"

// Conservative 4-byte stepping works for every architecturally allowed cache
// line size and does not assume the host's instruction/data line sizes match.
TEXT ·pureFlush(SB),NOSPLIT,$0-16
	MOVD base+0(FP), R0
	MOVD size+8(FP), R1
	ADD R0, R1, R1
	MOVD R0, R2
dc_loop:
	WORD $0xd50b7b22 // dc cvau, x2
	ADD $4, R2
	CMP R1, R2
	BLT dc_loop
	WORD $0xd5033b9f // dsb ish
ic_loop:
	WORD $0xd50b7520 // ic ivau, x0
	ADD $4, R0
	CMP R1, R0
	BLT ic_loop
	WORD $0xd5033b9f // dsb ish
	WORD $0xd5033fdf // isb
	RET
