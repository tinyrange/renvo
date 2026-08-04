//go:build !renvo && (linux || windows) && amd64

#include "textflag.h"

TEXT ·callJIT(SB),NOSPLIT,$0-56
	MOVQ entry+0(FP), AX
	MOVQ stackTop+8(FP), R10
	MOVQ argsData+16(FP), DI
	MOVQ argsLen+24(FP), SI
	MOVQ envData+32(FP), DX
	MOVQ envLen+40(FP), CX

	// The linked code owns these registers, including Go's g register R14.
	PUSHQ BX
	PUSHQ BP
	PUSHQ R12
	PUSHQ R13
	PUSHQ R14
	PUSHQ R15
	MOVQ SP, R11
	MOVQ R10, SP
	SUBQ $16, SP
	MOVQ R11, 8(SP)
	CALL AX
	MOVQ AX, R10
	MOVQ 8(SP), R11
	MOVQ R11, SP
	POPQ R15
	POPQ R14
	POPQ R13
	POPQ R12
	POPQ BP
	POPQ BX
	MOVQ R10, ret+48(FP)
	RET
