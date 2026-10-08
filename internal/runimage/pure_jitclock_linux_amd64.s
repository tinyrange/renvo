//go:build !renvo && linux && amd64

#include "textflag.h"

// Serialize prior code publication before timestamping its load record.
TEXT ·jitReadTSC(SB), NOSPLIT, $0-8
	LFENCE
	RDTSC
	SHLQ $32, DX
	ORQ DX, AX
	MOVQ AX, ret+0(FP)
	RET
