package main

func wideReturnConversionValue() int64 {
	var mapping map[string]int64
	zero, _ := mapping["missing"]
	copied := 3
	return zero + int64(copied+len(mapping))
}

func main() {
	if wideReturnConversionValue() != 3 {
		print("FAIL: narrow expression conversion in wide return\n")
		return
	}
	print("PASS\n")
}
