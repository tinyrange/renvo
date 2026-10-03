package main

import "unsafe"

type vectorBlock32 [32]byte

func copyVector32(data *[128]byte, source int, destination int) {
	src := (*vectorBlock32)(unsafe.Pointer(&data[source]))
	dst := (*vectorBlock32)(unsafe.Pointer(&data[destination]))
	*dst = *src
}

type vectorBlock47 [47]byte

func copyVector47(data *[128]byte, source int, destination int) {
	src := (*vectorBlock47)(unsafe.Pointer(&data[source]))
	dst := (*vectorBlock47)(unsafe.Pointer(&data[destination]))
	*dst = *src
}

type vectorBlock65 [65]byte

func copyVector65(data *[128]byte, source int, destination int) {
	src := (*vectorBlock65)(unsafe.Pointer(&data[source]))
	dst := (*vectorBlock65)(unsafe.Pointer(&data[destination]))
	*dst = *src
}

type vectorBlock96 [96]byte

func copyVector96(data *[128]byte, source int, destination int) {
	src := (*vectorBlock96)(unsafe.Pointer(&data[source]))
	dst := (*vectorBlock96)(unsafe.Pointer(&data[destination]))
	*dst = *src
}
func appMain() int {
	for test := 0; test < 4; test++ {
		size := 32
		if test == 1 {
			size = 47
		} else if test == 2 {
			size = 65
		} else if test == 3 {
			size = 96
		}
		for source := 0; source < 5; source++ {
			for destination := 0; destination < 5; destination++ {
				var data [128]byte
				var original [128]byte
				for i := 0; i < 128; i++ {
					data[i] = byte(i*37 + test)
					original[i] = data[i]
				}
				if test == 0 {
					copyVector32(&data, source, destination)
				} else if test == 1 {
					copyVector47(&data, source, destination)
				} else if test == 2 {
					copyVector65(&data, source, destination)
				} else {
					copyVector96(&data, source, destination)
				}
				for i := 0; i < 128; i++ {
					want := original[i]
					if i >= destination && i < destination+size {
						want = original[source+i-destination]
					}
					if data[i] != want {
						print("FAIL\n")
						return 1
					}
				}
			}
		}
	}
	print("PASS\n")
	return 0
}
