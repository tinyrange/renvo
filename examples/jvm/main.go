package main

import "java"

func main() {
	vm, ok := java.GetProperty("java.vm.name")
	if !ok {
		print("could not read java.vm.name\n")
		return
	}
	version, ok := java.CallStaticString(
		"java.lang.System", "getProperty", "java.version")
	if !ok {
		print("reflective Java call failed\n")
		return
	}
	print("Renvo on ")
	print(vm)
	print(" / Java ")
	print(version)
	print("\n")
}
