package main

func compileLinuxTarget(input []int, output int, target int) int {
	return compileTarget(input, output, target, 0)
}

func renvoLinuxSysWriteSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysWriteSeq
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysWriteSeq
		}
		return renvoOpenBSDAmd64SysWriteSeq
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 64
	}
	if renvoTargetArch == renvoArchArm {
		return 4
	}
	if renvoTargetArch == renvoArch386 {
		return 4
	}
	return 1
}

func renvoLinuxSysReadSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysReadSeq
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysReadSeq
		}
		return renvoOpenBSDAmd64SysReadSeq
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 63
	}
	if renvoTargetArch == renvoArchArm {
		return 3
	}
	if renvoTargetArch == renvoArch386 {
		return 3
	}
	return 0
}

func renvoLinuxSysReadAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysReadAt
		}
		if renvoTargetOS == renvoOSOpenBSD {
			return renvoOpenBSDAmd64SysReadAt
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysReadAt
		}
		return 17
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 67
	}
	if renvoTargetArch == renvoArchArm {
		return 180
	}
	if renvoTargetArch == renvoArch386 {
		return 180
	}
	return 17
}

func renvoLinuxSysWriteAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysWriteAt
		}
		if renvoTargetOS == renvoOSOpenBSD {
			return renvoOpenBSDAmd64SysWriteAt
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysWriteAt
		}
		return 18
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 68
	}
	if renvoTargetArch == renvoArchArm {
		return 181
	}
	if renvoTargetArch == renvoArch386 {
		return 181
	}
	return 18
}

func renvoLinuxSysOpen(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysOpen
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysOpen
		}
		return renvoOpenBSDAmd64SysOpen
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 56
	}
	if renvoTargetArch == renvoArchArm {
		return 5
	}
	if renvoTargetArch == renvoArch386 {
		return 5
	}
	return 2
}

func renvoLinuxSysClose(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysClose
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysClose
		}
		return renvoOpenBSDAmd64SysClose
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 57
	}
	if renvoTargetArch == renvoArchArm {
		return 6
	}
	if renvoTargetArch == renvoArch386 {
		return 6
	}
	return 3
}

func renvoLinuxSysFchmod(renvoTargetOS int, renvoTargetArch int) int {
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysFchmod
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysFchmod
		}
		return renvoOpenBSDAmd64SysFchmod
	}
	if renvoTargetArch == renvoArchAarch64 {
		return 52
	}
	if renvoTargetArch == renvoArchArm {
		return 94
	}
	if renvoTargetArch == renvoArch386 {
		return 94
	}
	return 91
}

func renvoHostedAmd64SysExit(renvoTargetOS int) int {
	if targetIsBSD(renvoTargetOS) {
		if renvoTargetOS == renvoOSFreeBSD {
			return renvoFreeBSDAmd64SysExit
		}
		if renvoTargetOS == renvoOSNetBSD {
			return renvoNetBSDAmd64SysExit
		}
		return renvoOpenBSDAmd64SysExit
	}
	return 60
}

func renvoAsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoNonNil(a)
	if a.c.renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRsiRax(a)
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRcx)
		return
	}
	if a.c.renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPrepareReadWriteBuf(a)
		return
	}
	if a.c.renvoTargetArch == renvoArchArm {
		renvoArmAsmPrepareReadWriteBuf(a)
		return
	}
	if a.c.renvoTargetArch == renvoArch386 {
		renvo386AsmPrepareReadWriteBuf(a)
		return
	}
	renvoAmd64AsmPrepareReadWriteBuf(a)
}

func renvoAsmMoveOffsetArg(a *renvoAsm) {
	renvoNonNil(a)
	if a.c.renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegR10, renvoWasm32RegRax)
		return
	}
	if a.c.renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMoveOffsetArg(a)
		return
	}
	if a.c.renvoTargetArch == renvoArchArm {
		renvoArmAsmMoveOffsetArg(a)
		return
	}
	if a.c.renvoTargetArch == renvoArch386 {
		renvo386AsmMoveOffsetArg(a)
		return
	}
	renvoAmd64AsmMoveOffsetArg(a)
}
