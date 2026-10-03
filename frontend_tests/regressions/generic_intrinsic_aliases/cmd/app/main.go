package main

import "fmt"

var calls int

func renvo_runtime_FmtPrintln() int { calls++; return 7 }
func renvo_runtime_Syscall() int    { return 11 }
func Id[T any](value T) T           { return value }

func Display(renvo_runtime_FmtPrintln func() int) {
	if Id(renvo_runtime_FmtPrintln)() != 17 {
		panic("callback parameter identity")
	}
	fmt.Println("PASS")
}

func main() {
	if Id(renvo_runtime_FmtPrintln)() != 7 || Id(renvo_runtime_Syscall)() != 11 {
		panic("authored global function identity")
	}
	renvo_runtime_FmtPrintln()
	if calls != 2 {
		panic("authored direct call identity")
	}
	renvo_runtime_FmtPrintln := func() int { return 17 }
	renvo_runtime_FmtPrintln_ := 19
	renvo_runtime_FmtPrintln__ := 23
	if Id(renvo_runtime_FmtPrintln()) != 17 || renvo_runtime_FmtPrintln_+renvo_runtime_FmtPrintln__ != 42 {
		panic("local callback and suffix identity")
	}
	RenvoGenericInstance_0 := 29
	RenvoGenericInstance_0_ := 31
	if Id(RenvoGenericInstance_0)+Id(RenvoGenericInstance_0_) != 60 {
		panic("specialization name capture")
	}
	Display(renvo_runtime_FmtPrintln)
}
