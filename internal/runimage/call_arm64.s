//go:build !renvo && (linux || windows || darwin) && arm64

#include "textflag.h"

TEXT ·callJIT(SB),NOSPLIT,$0-56
	MOVD entry+0(FP), R9
	MOVD stackTop+8(FP), R10
	MOVD argsData+16(FP), R0
	MOVD argsLen+24(FP), R1
	MOVD envData+32(FP), R2
	MOVD envLen+40(FP), R3
	MOVD RSP, R11
	MOVD R10, RSP
	SUB $16, RSP
	MOVD R11, 8(RSP)
	CALL (R9)
	MOVD R0, R10
	MOVD 8(RSP), R11
	MOVD R11, RSP
	MOVD R10, ret+48(FP)
	RET
