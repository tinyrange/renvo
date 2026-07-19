package main

const (
	renvoIotaSliceA = iota + 2
	renvoIotaSliceB
	renvoIotaSliceC
)

func appMain(args []string) int {
	xs := []int{renvoIotaSliceC, renvoIotaSliceA, renvoIotaSliceB}
	if xs[0] != 4 || xs[1] != 2 || xs[2] != 3 {
		print("RENVO-IOTA-014 slice literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
