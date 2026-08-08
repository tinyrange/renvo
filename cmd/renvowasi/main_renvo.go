//go:build renvo_wasi_frontend

package main

import (
	"renvo.dev/internal/driver"
	"renvo.dev/internal/unit"
)

func renvoWasiHexDigit(value byte) int {
	if value >= '0' && value <= '9' {
		return int(value - '0')
	}
	if value >= 'a' && value <= 'f' {
		return int(value-'a') + 10
	}
	if value >= 'A' && value <= 'F' {
		return int(value-'A') + 10
	}
	return -1
}

func renvoWasiDefinition(value string) (string, bool) {
	if len(value) != 64 {
		return "", false
	}
	definition := make([]byte, 32)
	for i := 0; i < len(definition); i++ {
		high := renvoWasiHexDigit(value[i*2])
		low := renvoWasiHexDigit(value[i*2+1])
		if high < 0 || low < 0 {
			return "", false
		}
		definition[i] = byte(high<<4 | low)
	}
	return string(definition), true
}

func renvoWasiPositiveDecimal(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	result := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		result = result*10 + int(value[i]-'0')
	}
	return result, true
}

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
	target := "wasi/wasm32"
	definition := ""
	descriptorVersion := 0
	tags := make([]string, 0, 4)
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			i++
			output = args[i]
		} else if args[i] == "-tags" && i+1 < len(args) {
			i++
			tags = append(tags, args[i])
		} else if args[i] == "-t" && i+1 < len(args) {
			i++
			target = args[i]
		} else if args[i] == "-target-definition" && i+1 < len(args) {
			i++
			var ok bool
			definition, ok = renvoWasiDefinition(args[i])
			if !ok {
				print("renvo: invalid target definition\n")
				return 2
			}
		} else if args[i] == "-target-version" && i+1 < len(args) {
			i++
			var ok bool
			descriptorVersion, ok = renvoWasiPositiveDecimal(args[i])
			if !ok {
				print("renvo: invalid target descriptor version\n")
				return 2
			}
		} else if args[i] != "-s" && args[i] != "-emit-unit" {
			packageArg = args[i]
		}
	}
	built := driver.BuildPackageUnitCompact(packageArg, target, tags, workDir, stdRoot, driver.RenvoFS{})
	if !built.Ok {
		print("renvo: frontend compilation failed\n")
		return 1
	}
	if definition != "" {
		bound, ok := unit.BindTarget(built.Unit, unit.TargetBinding{
			Target: target, Definition: definition, DescriptorVersion: descriptorVersion,
		})
		if !ok {
			print("renvo: could not bind custom target\n")
			return 1
		}
		built.Unit = bound
	}
	if !writeOutput(output, built.Unit) {
		print("renvo: could not write output\n")
		return 1
	}
	return 0
}
