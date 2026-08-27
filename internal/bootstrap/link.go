//go:build !renvo

package bootstrap

import (
	"fmt"
	"os"

	"renvo.dev/internal/elflink"
)

func objectLinkCommandRequested(args []string) bool {
	if len(args) < 3 || args[1] != "cc" {
		return false
	}
	hasObject, hasSource := false, false
	for i := 2; i < len(args); i++ {
		if hasSuffix(args[i], ".o") {
			hasObject = true
		}
		if hasSuffix(args[i], ".c") {
			hasSource = true
		}
	}
	return hasObject && !hasSource
}

func runObjectLinkCommand(args []string) int {
	output := "a.out"
	inputs := make([]elflink.Input, 0, 4)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "-o" && i+1 < len(args) {
			i++
			output = args[i]
			continue
		}
		if arg == "-t" && i+1 < len(args) {
			i++
			if args[i] != "linux/amd64" {
				fmt.Fprintf(os.Stderr, "renvo cc: object linking only supports linux/amd64, not %s\n", args[i])
				return 2
			}
			continue
		}
		if arg == "-s" || arg == "-static" || arg == "-nostdlib" {
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			fmt.Fprintf(os.Stderr, "renvo cc: unsupported linker option: %s\n", arg)
			return 2
		}
		data, err := os.ReadFile(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "renvo cc: could not read object %s: %v\n", arg, err)
			return 1
		}
		inputs = append(inputs, elflink.Input{Name: arg, Data: data})
	}
	image, linkError := elflink.Link(inputs)
	if linkError.Message != "" {
		if linkError.Input != "" {
			fmt.Fprintf(os.Stderr, "%s: error RENVO-LINK-001 (linker): %s\n", linkError.Input, linkError.Message)
		} else {
			fmt.Fprintf(os.Stderr, "renvo cc: error RENVO-LINK-001 (linker): %s\n", linkError.Message)
		}
		return 1
	}
	if err := os.WriteFile(output, image, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "renvo cc: could not write executable %s: %v\n", output, err)
		return 1
	}
	return 0
}

func hasSuffix(text, suffix string) bool {
	return len(text) >= len(suffix) && text[len(text)-len(suffix):] == suffix
}
