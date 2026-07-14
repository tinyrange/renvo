package main

import "path"

func main() {
	if path.Join("one", "two", "three", "four", "five") != "one/two/three/four/five" {
		return
	}
	parts := []string{"root", "child", "leaf"}
	if path.Join(parts...) != "root/child/leaf" {
		return
	}
	if path.Join("", "") != "" {
		return
	}
	print("PASS\n")
}
