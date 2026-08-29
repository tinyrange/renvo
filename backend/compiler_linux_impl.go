package main

func compileLinuxTarget(input []int, output int, target int) int {
	return compileTarget(input, output, target, 0)
}

func renvoLinuxSysWriteSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysWriteSeq
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysWriteSeq
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysWriteSeq
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysWriteSeq
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysWriteSeq
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysWriteSeq
		}
		return 0
	}
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysWriteSeq
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysWriteSeq
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysWriteSeq
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysWriteSeq
	}
	return renvoLinuxAmd64SysWriteSeq
}

func renvoLinuxSysReadSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysReadSeq
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysReadSeq
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysReadSeq
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysReadSeq
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysReadSeq
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysReadSeq
		}
		return 0
	}
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysReadSeq
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysReadSeq
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysReadSeq
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysReadSeq
	}
	return renvoLinuxAmd64SysReadSeq
}

func renvoLinuxSysReadAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysReadAt
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysReadAt
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysReadAt
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysReadAt
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysReadAt
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return renvoFreeBSDAmd64SysReadAt
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return renvoOpenBSDAmd64SysReadAt
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoNetBSDAmd64SysReadAt
		}
		return 0
	}
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
		return renvoLinuxAmd64SysReadAt
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysReadAt
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysReadAt
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysReadAt
	}
	return renvoLinuxAmd64SysReadAt
}

func renvoLinuxSysWriteAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysWriteAt
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysWriteAt
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysWriteAt
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysWriteAt
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysWriteAt
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return renvoFreeBSDAmd64SysWriteAt
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return renvoOpenBSDAmd64SysWriteAt
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoNetBSDAmd64SysWriteAt
		}
		return 0
	}
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
		return renvoLinuxAmd64SysWriteAt
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysWriteAt
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysWriteAt
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysWriteAt
	}
	return renvoLinuxAmd64SysWriteAt
}

func renvoLinuxSysOpen(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysOpen
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysOpen
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysOpen
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysOpen
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysOpen
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysOpen
		}
		return 0
	}
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysOpen
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysOpen
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysOpen
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysOpen
	}
	return renvoLinuxAmd64SysOpen
}

func renvoLinuxSysClose(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysClose
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysClose
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysClose
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysClose
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysClose
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysClose
		}
		return 0
	}
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysClose
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysClose
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysClose
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysClose
	}
	return renvoLinuxAmd64SysClose
}

func renvoLinuxSysFchmod(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysFchmod
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysFchmod
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return renvoLinux386SysFchmod
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return renvoLinuxAarch64SysFchmod
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return renvoLinuxArmSysFchmod
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysFchmod
		}
		return 0
	}
	if renvoTargetArch == renvoArchAmd64 && targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysFchmod
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoLinuxAarch64SysFchmod
	}
	if renvoTargetArch == renvoArchArm {
		return renvoLinuxArmSysFchmod
	}
	if renvoTargetArch == renvoArch386 {
		return renvoLinux386SysFchmod
	}
	return renvoLinuxAmd64SysFchmod
}

func renvoHostedAmd64SysExit(renvoTargetOS int) int {
	if renvoFixedTarget == renvoTargetWasiWasm32 || renvoFixedTarget == renvoTargetVM32 {
		return renvoResolvedLinuxAmd64SysExit
	}
	if renvoFixedTarget != 0 && renvoFixedTarget != renvoTargetWasiWasm32 && renvoFixedTarget != renvoTargetVM32 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return renvoLinuxAmd64SysExit
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 || renvoFixedTarget == renvoTargetOpenBSDAmd64 || renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return renvoBSDAmd64SysExit
		}
		return 0
	}
	if targetIsBSD(renvoTargetOS) {
		return renvoBSDAmd64SysExit
	}
	return renvoLinuxAmd64SysExit
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
