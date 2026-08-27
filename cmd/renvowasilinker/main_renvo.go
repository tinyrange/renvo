//go:build renvo_wasi_linker

package main

import "renvo.dev/internal/elflink"

func linkerReadFile(path string) ([]byte, bool) {
	fd := open(path, 0)
	if fd < 0 {
		return nil, false
	}
	out := make([]byte, 32768)
	used := 0
	for {
		if used == len(out) {
			next := make([]byte, len(out)*2)
			copy(next, out)
			out = next
		}
		n := read(fd, out[used:], -1)
		if n < 0 {
			close(fd)
			return nil, false
		}
		if n == 0 {
			break
		}
		used += n
	}
	close(fd)
	return out[:used], true
}

func linkerWriteFile(path string, data []byte) bool {
	fd := open(path, 578)
	if fd < 0 {
		return false
	}
	for at := 0; at < len(data); {
		n := write(fd, data[at:], -1)
		if n <= 0 {
			close(fd)
			return false
		}
		at += n
	}
	chmod(fd, 0755)
	return close(fd) == 0
}

func appMain(args []string, env []string) int {
	output := "a.out"
	inputs := make([]elflink.Input, 0, 4)
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			i++
			output = args[i]
			continue
		}
		data, ok := linkerReadFile(args[i])
		if !ok {
			print("renvo linker: could not read ")
			print(args[i])
			print("\n")
			return 1
		}
		inputs = append(inputs, elflink.Input{Name: args[i], Data: data})
	}
	image, err := elflink.Link(inputs)
	if err.Message != "" {
		print("renvo linker: error RENVO-LINK-001 (linker): ")
		print(err.Message)
		print("\n")
		return 1
	}
	if !linkerWriteFile(output, image) {
		print("renvo linker: could not write output\n")
		return 1
	}
	_ = env
	return 0
}
