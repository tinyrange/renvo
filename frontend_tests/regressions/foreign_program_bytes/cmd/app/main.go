package main

//renvo:compile -t linux/386 payloadMain
var payload []byte

func shared() int { return 42 }

func payloadMain() {
	if shared() != 42 {
		print("foreign failure\n")
	}
}

func main() {
	if shared() != 42 || len(payload) < 5 || payload[0] != 0x7f ||
		payload[1] != 'E' || payload[2] != 'L' || payload[3] != 'F' || payload[4] != 1 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
