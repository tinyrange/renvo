package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
)

// updateRuntimeNumbers projects the numeric runtime operations declared by RTG.
// The OS/ISA-only entrypoints are compatibility adapters for the existing
// compiler API: an unknown OS uses the Linux ABI for that ISA, and an unknown
// ISA uses linux/amd64. Fixed targets use their own descriptor, not that fallback.
func updateRuntimeNumbers(path string, descriptors []sourceDescriptor) error {
	const begin = "// BEGIN GENERATED RUNTIME NUMBERS"
	const end = "// END GENERATED RUNTIME NUMBERS"
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	start := bytes.Index(source, []byte(begin))
	finish := bytes.Index(source, []byte(end))
	if start < 0 || finish < start {
		return fmt.Errorf("runtime number markers missing from %s", path)
	}
	var backend []sourceDescriptor
	var fallback *sourceDescriptor
	for i := range descriptors {
		descriptor := descriptors[i]
		if descriptor.Constant != "" {
			backend = append(backend, descriptor)
		}
		if descriptor.Name == "linux/amd64" {
			fallback = &descriptors[i]
		}
	}
	if fallback == nil {
		return fmt.Errorf("runtime compatibility API requires linux/amd64")
	}
	sort.Slice(backend, func(i, j int) bool { return backend[i].BackendID < backend[j].BackendID })
	var out bytes.Buffer
	out.WriteString(begin + "\n// Code generated from RTG runtime operations; DO NOT EDIT.\n")
	for _, operation := range []struct{ name, suffix string }{
		{"read", "ReadSeq"}, {"write", "WriteSeq"}, {"read_at", "ReadAt"},
		{"write_at", "WriteAt"}, {"open", "Open"}, {"close", "Close"},
		{"chmod", "Fchmod"}, {"exit", "Exit"},
	} {
		defaultNumber, ok := fallback.RuntimeNumbers[operation.name]
		if !ok {
			return fmt.Errorf("linux/amd64 has no %s runtime number", operation.name)
		}
		hostedExit := operation.name == "exit"
		if hostedExit {
			out.WriteString("func renvoHostedAmd64SysExit(renvoTargetOS int) int {\n")
		} else {
			fmt.Fprintf(&out, "func renvoLinuxSys%s(renvoTargetOS int, renvoTargetArch int) int {\n", operation.suffix)
		}
		out.WriteString("if renvoFixedTarget != 0 {\n")
		for _, descriptor := range backend {
			number, present := descriptor.RuntimeNumbers[operation.name]
			if hostedExit {
				// Virtual compiler profiles historically expose the host-compatible exit
				// number even though virtual execution uses the runtime's exit emitter.
				if descriptor.Family == "structured32" {
					number, present = defaultNumber, true
				} else if descriptor.ISAID != fallback.ISAID {
					present = false
				}
			}
			if present {
				fmt.Fprintf(&out, "if renvoFixedTarget == %s { return %d }\n", descriptor.Constant, number)
			}
		}
		out.WriteString("return 0\n}\n")
		// Exact non-Linux numeric ABIs take precedence over the legacy Linux fallback.
		for _, descriptor := range backend {
			number, present := descriptor.RuntimeNumbers[operation.name]
			if !present || descriptor.OSID == fallback.OSID {
				continue
			}
			legacyNumber := defaultNumber
			if !hostedExit {
				for _, linux := range backend {
					if value, ok := linux.RuntimeNumbers[operation.name]; ok && linux.OSID == fallback.OSID && linux.ISAID == descriptor.ISAID {
						legacyNumber = value
					}
				}
			}
			if number == legacyNumber {
				continue
			}
			if hostedExit {
				if descriptor.ISAID == fallback.ISAID {
					fmt.Fprintf(&out, "if renvoTargetOS == %d { return %d }\n", descriptor.OSID, number)
				}
			} else {
				fmt.Fprintf(&out, "if renvoTargetOS == %d && renvoTargetArch == %d { return %d }\n", descriptor.OSID, descriptor.ISAID, number)
			}
		}
		if !hostedExit {
			for _, descriptor := range backend {
				number, present := descriptor.RuntimeNumbers[operation.name]
				if present && descriptor.OSID == fallback.OSID && descriptor.ISAID != fallback.ISAID {
					fmt.Fprintf(&out, "if renvoTargetArch == %d { return %d }\n", descriptor.ISAID, number)
				}
			}
		}
		fmt.Fprintf(&out, "return %d\n}\n", defaultNumber)
	}
	out.WriteString(end)
	result := append([]byte(nil), source[:start]...)
	result = append(result, out.Bytes()...)
	result = append(result, source[finish+len(end):]...)
	return writeFormatted(path, result)
}
