package main

var stringStorageOrder int

func storageOperand(number int, value string) string {
	stringStorageOrder = stringStorageOrder*10 + number
	return value
}

func appMain() int {
	value := storageOperand(1, "abc") + storageOperand(2, "") + storageOperand(3, "\x00xyz")
	if stringStorageOrder != 123 || value != "abc\x00xyz" {
		panic("concatenation order")
	}
	bytes := []byte(value)
	saved := string(bytes)
	bytes[0] = 'X'
	if saved != "abc\x00xyz" || value != saved || string(bytes) != "Xbc\x00xyz" {
		panic("conversion ownership")
	}
	stringStorageOrder = 0
	joined := storageOperand(1, value+"!") + (storageOperand(2, "left") + storageOperand(3, "right"))
	if stringStorageOrder != 123 || joined != "abc\x00xyz!leftright" {
		panic("nested operands")
	}
	print("PASS\n")
	return 0
}
