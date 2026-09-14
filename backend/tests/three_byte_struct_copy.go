package main

type rgbCopy struct{ red, green, blue uint8 }

func encodeRGBCopy(pixels []rgbCopy, data []byte) {
	for i := 0; i < len(pixels); i++ {
		pixel := pixels[i]
		data[i*3] = pixel.green
		data[i*3+1] = pixel.red
		data[i*3+2] = pixel.blue
	}
}

func appMain(args []string) int {
	pixels := [2]rgbCopy{{blue: 24}, {red: 17, green: 33, blue: 65}}
	var data [6]byte
	encodeRGBCopy(pixels[:], data[:])
	if data[0] != 0 || data[1] != 0 || data[2] != 24 || data[3] != 33 || data[4] != 17 || data[5] != 65 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
