package main

import "testing"

func TestTargetCoreTablesCoverEveryTarget(t *testing.T) {
	want := rtgTargetDarwinArm64 + 1
	if len(rtgTargetOSTable) != want || len(rtgTargetArchTable) != want || len(rtgTargetIntBitsTable) != want {
		t.Fatalf("core target table lengths = os:%d arch:%d int:%d, want %d", len(rtgTargetOSTable), len(rtgTargetArchTable), len(rtgTargetIntBitsTable), want)
	}
}

func TestSetTargetDerivesStateFromTargetProfile(t *testing.T) {
	savedFixed := rtgCompilerFixedTarget
	savedTarget := rtgCurrentTarget
	savedOS := rtgTargetOS
	savedArch := rtgTargetArch
	savedIntSize := rtgNativeIntSize
	defer func() {
		rtgCompilerFixedTarget = savedFixed
		rtgCurrentTarget = savedTarget
		rtgTargetOS = savedOS
		rtgTargetArch = savedArch
		rtgNativeIntSize = savedIntSize
	}()

	targets := []int{
		rtgTargetLinuxAmd64,
		rtgTargetLinux386,
		rtgTargetLinuxAarch64,
		rtgTargetLinuxArm,
		rtgTargetWindowsAmd64,
		rtgTargetWindows386,
		rtgTargetWasiWasm32,
		rtgTargetDarwinArm64,
	}
	rtgCompilerFixedTarget = 0
	for _, target := range targets {
		profile, ok := rtgProfileForTarget(target)
		if !ok {
			t.Fatalf("target %d has no profile", target)
		}
		rtgSetTarget(target)
		if rtgCurrentTarget != target || rtgTargetOS != profile.os || rtgTargetArch != profile.arch || rtgNativeIntSize != profile.intBits/8 {
			t.Fatalf("target %d state = target:%d os:%d arch:%d int:%d, profile = %#v", target, rtgCurrentTarget, rtgTargetOS, rtgTargetArch, rtgNativeIntSize, profile)
		}
	}
}

func TestSetTargetUsesFixedTargetProfile(t *testing.T) {
	savedFixed := rtgCompilerFixedTarget
	savedTarget := rtgCurrentTarget
	savedOS := rtgTargetOS
	savedArch := rtgTargetArch
	savedIntSize := rtgNativeIntSize
	defer func() {
		rtgCompilerFixedTarget = savedFixed
		rtgCurrentTarget = savedTarget
		rtgTargetOS = savedOS
		rtgTargetArch = savedArch
		rtgNativeIntSize = savedIntSize
	}()

	rtgCompilerFixedTarget = rtgTargetWindows386
	rtgSetTarget(rtgTargetLinuxAmd64)
	profile, _ := rtgProfileForTarget(rtgTargetWindows386)
	if rtgCurrentTarget != profile.target || rtgTargetOS != profile.os || rtgTargetArch != profile.arch || rtgNativeIntSize != profile.intBits/8 {
		t.Fatalf("fixed target state did not come from profile: target:%d os:%d arch:%d int:%d", rtgCurrentTarget, rtgTargetOS, rtgTargetArch, rtgNativeIntSize)
	}
}
