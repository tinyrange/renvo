package main

import "strconv"

func main() {
	price, err := strconv.ParseFloat("0.02", 64)
	limit, limitErr := strconv.ParseFloat("0.002", 64)
	if err != nil || limitErr != nil {
		print("FAIL: price parse\n")
		return
	}
	estimated := price * float64(125000) / 1000000
	if estimated == 0 || estimated <= limit {
		print("FAIL: cost limit\n")
		return
	}
	starlark := strconv.FormatFloat(0.2+0.02, 'g', -1, 64)
	roundTrip, roundTripErr := strconv.ParseFloat(starlark, 64)
	if roundTripErr != nil || roundTrip != 0.22 {
		print("FAIL: Starlark float round trip\n")
		return
	}
	print("PASS\n")
}
