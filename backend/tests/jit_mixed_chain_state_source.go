package main

// Supplemental parsed-source call-state regression. The failing generated-RFE
// ABI case, including mixed entry families and exact scheduling/fault prefixes,
// is internal/rfe/runtime/native_mixed_family_chain_test.go.
func jitMixedStateLeaf(state []int) int {
	if state[1] == 0 {
		return state[2]
	}
	return state[3]
}
func jitMixedStateCaller(state []int) {
	a, b, c, d, e, f := state[5], state[6], state[7], state[8], state[9], state[10]
	state[0] += 7
	result := jitMixedStateLeaf(state)
	state[1] = result
	state[5], state[6], state[7] = a+1, b+2, c+3
	state[8], state[9], state[10] = d+4, e+5, f+6
}
func appMain(args []string) int {
	state := make([]int, 12)
	state[2], state[3] = 23, 41
	for i := 0; i < 1000; i++ {
		state[1] = i & 1
		jitMixedStateCaller(state)
		want := 23
		if i&1 != 0 {
			want = 41
		}
		if state[1] != want {
			print("FAIL\n")
			return 1
		}
	}
	if state[0] != 7000 {
		print("FAIL\n")
		return 1
	}
	for i := 5; i < 11; i++ {
		if state[i] != 1000*(i-4) {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
