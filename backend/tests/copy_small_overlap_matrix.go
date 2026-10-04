package main

func appMain(args []string) int {
	var data [192]byte
	var before [192]byte
	for size := 0; size <= 144; size++ {
		for source := 0; source < 24; source++ {
			for destination := 0; destination < 24; destination++ {
				for i := 0; i < len(data); i++ {
					data[i] = byte(i + 1)
					before[i] = byte(i + 1)
				}
				if copy(data[destination:destination+size], data[source:source+size]) != size {
					return 1
				}
				for i := 0; i < len(data); i++ {
					want := before[i]
					if i >= destination && i < destination+size {
						want = before[source+i-destination]
					}
					if data[i] != want {
						return 2
					}
				}
			}
		}
	}
	print("PASS\n")
	return 0
}
