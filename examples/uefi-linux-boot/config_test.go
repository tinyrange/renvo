package main

import "testing"

func TestParseConfig(t *testing.T) {
	config, problem := parseConfig([]byte("# boot\r\nkernel = \\vmlinuz\ninitramfs=\\initramfs\ncmdline=console=ttyS0 quiet\n"))
	if problem != "" {
		t.Fatal(problem)
	}
	if config.Kernel != "\\vmlinuz" || config.Initramfs != "\\initramfs" || config.Command != "console=ttyS0 quiet" {
		t.Fatalf("config = %#v", config)
	}
}

func TestParseConfigErrors(t *testing.T) {
	tests := []struct{ input, want string }{
		{"initramfs=x\n", "config.txt: kernel is required"},
		{"kernel=x\n", "config.txt: initramfs is required"},
		{"kernel=x\nwat=y\n", "config.txt line 2: unknown setting wat"},
		{"kernel\n", "config.txt line 1: expected name=value"},
	}
	for _, test := range tests {
		_, got := parseConfig([]byte(test.input))
		if got != test.want {
			t.Errorf("parseConfig(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
