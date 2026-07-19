package main

type pair struct {
	real int
	imag int
}

func appMain(args []string) int {
	var terms []pair
	terms = append(terms, pair{real: 3, imag: 4})
	terms = append(terms, pair{real: 5, imag: -2})
	if len(terms) != 2 {
		print("RENVO-0842 pair append len failed\n")
		return 1
	}
	if terms[0].real+terms[1].imag != 1 {
		print("RENVO-0842 pair arithmetic failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
