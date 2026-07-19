package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const raw = `hello`
const legacyOctal = 077
const modernOctal = 0o77

const (
	bitA = 1 << iota
	bitB
)

type arrayBox struct {
	values [2]int `json:"values"`
}

type table map[string]int

func mutateArray(values [2]int) [2]int {
	values[0] = 9
	return values
}

func namedResult() (value int) {
	value = 5
	return
}

func main() {
	if raw != "hello" {
		print("FAIL raw string\n")
		return
	}
	if legacyOctal != 63 || modernOctal != 63 {
		print("FAIL octal\n")
		return
	}
	if bitA != 1 || bitB != 2 {
		print("FAIL iota\n")
		return
	}
	if 0x1.8p+1 != 3.0 {
		print("FAIL hex float\n")
		return
	}

	original := [2]int{1, 2}
	changed := mutateArray(original)
	grid := [2][2]int{{3, 4}, {5, 6}}
	boxed := arrayBox{values: [2]int{7, 8}}
	if original[0] != 1 || changed[0] != 9 || len(changed) != 2 || cap(changed) != 2 || grid[1][1] != 6 || boxed.values[1] != 8 {
		print("FAIL arrays\n")
		return
	}

	values := table{"a": 1}
	values["b"] = 2
	made := make(map[string]int)
	made["c"] = 3
	mapSum := 0
	for _, value := range values {
		mapSum += value
	}
	if values["a"] != 1 {
		print("FAIL map index\n")
		return
	}
	if values["b"] != 2 || made["c"] != 3 || mapSum != 3 {
		print("FAIL maps\n")
		return
	}

	sliceSum := 0
	for _, value := range []int{1, 2, 3} {
		sliceSum += value
	}
	window := make([]int, 0, 4)
	window = append(window, 4, 5, 6)
	window = window[0:2:3]
	if sliceSum != 6 || len(window) != 2 || cap(window) != 3 || window[1] != 5 || namedResult() != 5 {
		print("FAIL slices\n")
		return
	}

	sorted := []int{3, 1, 2}
	sort.Ints(sorted)
	err := errors.New("problem")
	if sorted[0] != 1 || sorted[2] != 3 || fmt.Sprintf("%d", 12) != "12" || strings.ReplaceAll("a-b-a", "a", "x") != "x-b-x" || err.Error() != "problem" {
		print("FAIL std\n")
		return
	}
	print("PASS\n")
}
