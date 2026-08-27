//go:build renvo && uefi && amd64

package uefi

func imageHandle() uintptr
func systemTable() uintptr
func call0(function uintptr) uintptr
func call1(function, a0 uintptr) uintptr
func call2(function, a0, a1 uintptr) uintptr
func call3(function, a0, a1, a2 uintptr) uintptr
func call4(function, a0, a1, a2, a3 uintptr) uintptr
func call5(function, a0, a1, a2, a3, a4 uintptr) uintptr
