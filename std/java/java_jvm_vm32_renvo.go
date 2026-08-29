//go:build renvo && jvm && vm32

// Package java exposes the explicit Java boundary supplied by the JVM RBE.
// This v1 surface copies strings and does not expose JVM references.
package java

import "syscall"

const (
	hostSystemProperty   = 0x4a01
	hostCallStaticString = 0x4a02
)

func GetProperty(name string) (string, bool) {
	buffer := make([]byte, 4096)
	n := syscall.JVMCall(hostSystemProperty, name, buffer)
	if n < 0 || n > len(buffer) {
		return "", false
	}
	return string(buffer[:n]), true
}

// CallStaticString invokes a public static Java method with exactly one
// java.lang.String parameter. The result is converted with String.valueOf.
func CallStaticString(className, methodName, argument string) (string, bool) {
	request := make([]byte, 6+len(className)+len(methodName)+len(argument))
	put16(request, 0, len(className))
	put16(request, 2, len(methodName))
	put16(request, 4, len(argument))
	at := 6
	at = copyText(request, at, className)
	at = copyText(request, at, methodName)
	copyText(request, at, argument)
	buffer := make([]byte, 4096)
	n := syscall.JVMCall(hostCallStaticString, string(request), buffer)
	if n < 0 || n > len(buffer) {
		return "", false
	}
	return string(buffer[:n]), true
}

func put16(data []byte, at, value int) {
	data[at] = byte(value)
	data[at+1] = byte(value >> 8)
}

func copyText(data []byte, at int, value string) int {
	for i := 0; i < len(value); i++ {
		data[at+i] = value[i]
	}
	return at + len(value)
}
