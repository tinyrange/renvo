//go:build jvm

package main

//renvo:compile -t jvm/vm32 payloadMain
var payload []byte

func shared() int { return 42 }

func payloadMain() {
	if shared() != 42 {
		print("foreign failure\n")
	}
}

func main() {
	if shared() != 42 || len(payload) < 8 || payload[0] != 0xca ||
		payload[1] != 0xfe || payload[2] != 0xba || payload[3] != 0xbe ||
		payload[4] != 0 || payload[5] != 0 || payload[6] != 0 || payload[7] != 49 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
