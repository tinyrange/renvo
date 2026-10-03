package main

func compileLinuxTarget(input []int, output int, target int) int {
	return compileTarget(input, output, target, 0)
}

// BEGIN GENERATED RUNTIME NUMBERS
// Code generated from RTG runtime operations; DO NOT EDIT.
func renvoLinuxSysReadSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 0
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 3
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 63
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 3
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 0
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 0
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 3
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 3
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 3
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 3
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 3
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 3
	}
	if renvoTargetArch == 2 {
		return 3
	}
	if renvoTargetArch == 3 {
		return 63
	}
	if renvoTargetArch == 4 {
		return 3
	}
	return 0
}
func renvoLinuxSysWriteSeq(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 1
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 4
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 64
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 4
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 1
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 1
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 4
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 4
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 4
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 4
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 4
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 4
	}
	if renvoTargetArch == 2 {
		return 4
	}
	if renvoTargetArch == 3 {
		return 64
	}
	if renvoTargetArch == 4 {
		return 4
	}
	return 1
}
func renvoLinuxSysReadAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 17
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 180
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 67
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 180
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 17
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 17
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 475
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 169
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 173
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 475
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 169
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 173
	}
	if renvoTargetArch == 2 {
		return 180
	}
	if renvoTargetArch == 3 {
		return 67
	}
	if renvoTargetArch == 4 {
		return 180
	}
	return 17
}
func renvoLinuxSysWriteAt(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 18
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 181
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 68
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 181
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 18
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 18
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 476
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 170
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 174
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 476
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 170
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 174
	}
	if renvoTargetArch == 2 {
		return 181
	}
	if renvoTargetArch == 3 {
		return 68
	}
	if renvoTargetArch == 4 {
		return 181
	}
	return 18
}
func renvoLinuxSysOpen(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 2
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 5
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 56
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 5
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 2
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 2
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 5
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 5
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 5
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 5
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 5
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 5
	}
	if renvoTargetArch == 2 {
		return 5
	}
	if renvoTargetArch == 3 {
		return 56
	}
	if renvoTargetArch == 4 {
		return 5
	}
	return 2
}
func renvoLinuxSysClose(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 3
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 6
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 57
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 6
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 3
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 3
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 6
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 6
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 6
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 6
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 6
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 6
	}
	if renvoTargetArch == 2 {
		return 6
	}
	if renvoTargetArch == 3 {
		return 57
	}
	if renvoTargetArch == 4 {
		return 6
	}
	return 3
}
func renvoLinuxSysFchmod(renvoTargetOS int, renvoTargetArch int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 91
		}
		if renvoFixedTarget == renvoTargetLinux386 {
			return 94
		}
		if renvoFixedTarget == renvoTargetLinuxAarch64 {
			return 52
		}
		if renvoFixedTarget == renvoTargetLinuxArm {
			return 94
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 91
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 91
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 124
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 124
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 124
		}
		return 0
	}
	if renvoTargetOS == 7 && renvoTargetArch == 1 {
		return 124
	}
	if renvoTargetOS == 8 && renvoTargetArch == 1 {
		return 124
	}
	if renvoTargetOS == 9 && renvoTargetArch == 1 {
		return 124
	}
	if renvoTargetArch == 2 {
		return 94
	}
	if renvoTargetArch == 3 {
		return 52
	}
	if renvoTargetArch == 4 {
		return 94
	}
	return 91
}
func renvoHostedAmd64SysExit(renvoTargetOS int) int {
	if renvoFixedTarget != 0 {
		if renvoFixedTarget == renvoTargetLinuxAmd64 {
			return 60
		}
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			return 60
		}
		if renvoFixedTarget == renvoTargetVM32 {
			return 60
		}
		if renvoFixedTarget == renvoTargetFreeBSDAmd64 {
			return 1
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 {
			return 1
		}
		if renvoFixedTarget == renvoTargetNetBSDAmd64 {
			return 1
		}
		return 0
	}
	if renvoTargetOS == 7 {
		return 1
	}
	if renvoTargetOS == 8 {
		return 1
	}
	if renvoTargetOS == 9 {
		return 1
	}
	return 60
}

// END GENERATED RUNTIME NUMBERS
