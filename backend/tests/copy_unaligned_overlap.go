package main

func appMain(args []string) int {
	for size := 0; size <= 65; size++ {
		for source := 0; source < 9; source++ {
			for dest := 0; dest < 9; dest++ {
				var data [96]byte
				var original [96]byte
				for i := 0; i < len(data); i++ {
					data[i] = byte(i*37 + size)
					original[i] = data[i]
				}
				n := copy(data[dest:dest+size], data[source:source+size])
				if n != size {
					print("FAIL count\n")
					return 1
				}
				for i := 0; i < len(data); i++ {
					want := original[i]
					if i >= dest && i < dest+size {
						want = original[source+i-dest]
					}
					if data[i] != want {
						print("FAIL copy\n")
						return 1
					}
				}
			}
		}
	}
	print("PASS\n")
	return 0
}
