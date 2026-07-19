package main

import "testing"

func TestTargetCoreTablesCoverEveryTarget(t *testing.T) {
	want := renvoTargetWindowsArm64 + 1
	if len(targetOSTable) != want || len(targetArchTable) != want || len(renvoTargetIntBitsTable) != want {
		t.Fatalf("core target table lengths = os:%d arch:%d int:%d, want %d", len(targetOSTable), len(targetArchTable), len(renvoTargetIntBitsTable), want)
	}
}

func TestSetTargetDerivesStateFromTargetProfile(t *testing.T) {
	savedFixed := compilerFixedTarget
	savedTarget := currentTarget
	savedOS := targetOS
	savedArch := targetArch
	savedIntSize := renvoNativeIntSize
	defer func() {
		compilerFixedTarget = savedFixed
		currentTarget = savedTarget
		targetOS = savedOS
		targetArch = savedArch
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
		renvoTargetDarwinArm64,
	}
	compilerFixedTarget = 0
	for _, target := range targets {
		profile, ok := renvoProfileForTarget(target)
		if !ok {
			t.Fatalf("target %d has no profile", target)
		}
		renvoSetTarget(target)
		if currentTarget != target || targetOS != profile.os || targetArch != profile.arch || renvoNativeIntSize != profile.intBits/8 {
			t.Fatalf("target %d state = target:%d os:%d arch:%d int:%d, profile = %#v", target, currentTarget, targetOS, targetArch, renvoNativeIntSize, profile)
		}
	}
}

func TestSetTargetUsesFixedTargetProfile(t *testing.T) {
	savedFixed := compilerFixedTarget
	savedTarget := currentTarget
	savedOS := targetOS
	savedArch := targetArch
	savedIntSize := renvoNativeIntSize
	defer func() {
		compilerFixedTarget = savedFixed
		currentTarget = savedTarget
		targetOS = savedOS
		targetArch = savedArch
		renvoNativeIntSize = savedIntSize
	}()

	compilerFixedTarget = renvoTargetWindows386
	renvoSetTarget(renvoTargetLinuxAmd64)
	profile, _ := renvoProfileForTarget(renvoTargetWindows386)
	if currentTarget != profile.target || targetOS != profile.os || targetArch != profile.arch || renvoNativeIntSize != profile.intBits/8 {
		t.Fatalf("fixed target state did not come from profile: target:%d os:%d arch:%d int:%d", currentTarget, targetOS, targetArch, renvoNativeIntSize)
	}
}
