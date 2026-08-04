//go:build !renvo && (linux || windows || (darwin && arm64))

package runimage

import "unsafe"

const jitStackSize = 8 << 20

func jitArguments(script string, args []string, env []string) ([][]byte, []uintptr, [][]byte, []uintptr) {
	programArgs := make([]string, 1, len(args)+1)
	programArgs[0] = script
	programArgs = append(programArgs, args...)
	argStorage, argWords := jitStringWords(programArgs)
	envStorage, envWords := jitStringWords(env)
	return argStorage, argWords, envStorage, envWords
}

func jitStringWords(values []string) ([][]byte, []uintptr) {
	storage := make([][]byte, len(values))
	stride := 16 / int(unsafe.Sizeof(uintptr(0)))
	words := make([]uintptr, len(values)*stride)
	for i := range values {
		storage[i] = []byte(values[i])
		if len(storage[i]) != 0 {
			words[i*stride] = uintptr(unsafe.Pointer(&storage[i][0]))
		}
		words[i*stride+stride/2] = uintptr(len(storage[i]))
	}
	return storage, words
}

func jitWordPointer(words []uintptr) uintptr {
	if len(words) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&words[0]))
}

func pageAlign(value int, pageSize int) int {
	return (value + pageSize - 1) &^ (pageSize - 1)
}

func pageFloor(value int, pageSize int) int {
	return value &^ (pageSize - 1)
}
