//go:build !renvo

#include "textflag.h"

TEXT ·callPure(SB),NOSPLIT,$0-24
	MOVD entry+0(FP), R9
	MOVD state+8(FP), R0
	MOVD stackTop+16(FP), R10
	MOVD RSP, R11
	MOVD R10, RSP
	SUB $16, RSP
	MOVD R11, 8(RSP)
	CALL (R9)
	MOVD 8(RSP), R11
	MOVD R11, RSP
	RET
