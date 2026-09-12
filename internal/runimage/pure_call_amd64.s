//go:build !renvo

#include "textflag.h"

TEXT ·callPure(SB),NOSPLIT,$0-24
	MOVQ entry+0(FP), R10
	MOVQ state+8(FP), AX
	MOVQ stackTop+16(FP), R11
	MOVQ SP, CX
	MOVQ R11, SP
	SUBQ $16, SP
	MOVQ CX, 8(SP)
	CALL R10
	MOVQ 8(SP), CX
	MOVQ CX, SP
	RET
