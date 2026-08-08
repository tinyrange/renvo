//go:build renvo_wasi_frontend

package main

import "renvo.dev/internal/driver"

func writeOutput(path string, data []byte) bool {
	fd := open(path, 578)
	if fd < 0 {
		return false
	}
	written := 0
	for written < len(data) {
		n := write(fd, data[written:], -1)
		if n <= 0 {
			close(fd)
			return false
		}
		written += n
	}
	if chmod(fd, 0644) != 0 {
		close(fd)
		return false
	}
	return close(fd) == 0
}

func appMain(args []string, env []string) int {
	workDir := "."
	stdRoot := "./std"
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) > 4 && item[:4] == "PWD=" {
			workDir = item[4:]
		}
		if len(item) > 14 && item[:14] == "RENVO_STDROOT=" {
			stdRoot = item[14:]
		}
	}
	output := "a.unit"
	packageArg := "."
	tags := make([]string, 0, 4)
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			i++
			output = args[i]
		} else if args[i] == "-tags" && i+1 < len(args) {
			i++
			tags = append(tags, args[i])
		} else if args[i] != "-s" && args[i] != "-emit-unit" {
			packageArg = args[i]
		}
	}
	built := driver.BuildPackageUnitCompact(packageArg, "wasi/wasm32", tags, workDir, stdRoot, driver.RenvoFS{})
	if !built.Ok {
		print("renvo: frontend compilation failed\n")
		return 1
	}
	if !writeOutput(output, built.Unit) {
		print("renvo: could not write output\n")
		return 1
	}
	return 0
}
