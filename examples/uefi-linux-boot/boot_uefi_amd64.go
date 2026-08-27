//go:build renvo && uefi && amd64

package main

func zeroMemory(address, size uintptr)
func enterLinux64(transition uintptr)
