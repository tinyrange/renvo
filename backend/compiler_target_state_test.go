package main

import "testing"

func TestTargetCoreTablesCoverEveryTarget(t *testing.T) {
	want := renvoTargetVM32 + 1
	if got := len(targetOSTable); got != want {
		t.Fatalf("OS target table length = %d, want %d", got, want)
	}
	if got := len(targetArchTable); got != want {
		t.Fatalf("architecture target table length = %d, want %d", got, want)
	}
	if got := len(renvoTargetIntBitsTable); got != want {
		t.Fatalf("integer-width target table length = %d, want %d", got, want)
	}
}

func TestSetTargetDerivesStateFromTargetProfile(t *testing.T) {
	savedFixed := renvoFixedTarget
	savedTarget := renvoTarget
	savedOS := renvoTargetOS
	savedArch := renvoTargetArch
	savedIntSize := renvoNativeIntSize
	defer func() {
		renvoFixedTarget = savedFixed
		renvoTarget = savedTarget
		renvoTargetOS = savedOS
		renvoTargetArch = savedArch
		renvoNativeIntSize = savedIntSize
	}()

	targets := []int{
		renvoTargetLinuxAmd64,
		renvoTargetLinux386,
		renvoTargetLinuxAarch64,
		renvoTargetLinuxArm,
		renvoTargetWindowsAmd64,
		renvoTargetWindows386,
		renvoTargetWindowsArm64,
		renvoTargetWasiWasm32,
		renvoTargetVM32,
		renvoTargetDarwinArm64,
	}
	renvoFixedTarget = 0
	for _, target := range targets {
		profile, ok := renvoProfileForTarget(target)
		if !ok {
			t.Fatalf("target %d has no profile", target)
		}
		renvoSetTarget(target)
		if renvoTarget != target || renvoTargetOS != profile.os || renvoTargetArch != profile.arch || renvoNativeIntSize != profile.intBits/8 {
			t.Fatalf("target %d state = target:%d os:%d arch:%d int:%d, profile = %#v", target, renvoTarget, renvoTargetOS, renvoTargetArch, renvoNativeIntSize, profile)
		}
	}
}

func TestSetTargetUsesFixedTargetProfile(t *testing.T) {
	savedFixed := renvoFixedTarget
	savedTarget := renvoTarget
	savedOS := renvoTargetOS
	savedArch := renvoTargetArch
	savedIntSize := renvoNativeIntSize
	defer func() {
		renvoFixedTarget = savedFixed
		renvoTarget = savedTarget
		renvoTargetOS = savedOS
		renvoTargetArch = savedArch
		renvoNativeIntSize = savedIntSize
	}()

	renvoFixedTarget = renvoTargetWindows386
	renvoSetTarget(renvoTargetLinuxAmd64)
	profile, _ := renvoProfileForTarget(renvoTargetWindows386)
	if renvoTarget != profile.target || renvoTargetOS != profile.os || renvoTargetArch != profile.arch || renvoNativeIntSize != profile.intBits/8 {
		t.Fatalf("fixed target state did not come from profile: target:%d os:%d arch:%d int:%d", renvoTarget, renvoTargetOS, renvoTargetArch, renvoNativeIntSize)
	}
}
