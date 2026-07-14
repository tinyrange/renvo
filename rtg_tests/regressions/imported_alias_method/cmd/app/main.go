package main

import "example.com/rtgtests/regressions/imported_alias_method/resource"

func main() {
	image := resource.NewImage()
	image.Destroy()
	if image.Value != 0 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
