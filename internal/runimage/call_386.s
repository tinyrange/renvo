//go:build !renvo && (linux || windows) && 386

#include "textflag.h"

TEXT ·callJIT(SB),NOSPLIT,$16-28
	MOVL BX, 0(SP)
	MOVL BP, 4(SP)
	MOVL SI, 8(SP)
	MOVL DI, 12(SP)
	MOVL entry+0(FP), AX
	MOVL stackTop+4(FP), BP
	MOVL argsData+8(FP), BX
	MOVL argsLen+12(FP), CX
	MOVL envData+16(FP), SI
	MOVL envLen+20(FP), DI
	MOVL SP, DX
	MOVL BP, SP
	SUBL $24, SP
	MOVL BX, 0(SP)
	MOVL CX, 4(SP)
	MOVL SI, 8(SP)
	MOVL DI, 12(SP)
	MOVL DX, 20(SP)
	CALL AX
	MOVL AX, CX
	MOVL 20(SP), DX
	MOVL DX, SP
	MOVL 12(SP), DI
	MOVL 8(SP), SI
	MOVL 4(SP), BP
	MOVL 0(SP), BX
	MOVL CX, ret+24(FP)
	RET
