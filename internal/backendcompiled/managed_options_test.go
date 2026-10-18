//go:build !renvo

package backendcompiled

import "testing"

func TestManagedHeapAPIBudgetAndOutputPolicy(t *testing.T) {
	source := []byte("package main;func appMain()int{return 0}")
	for _, tc := range []struct {
		name, target string
		options      RenvoCompileOptions
		ok           bool
	}{
		{"managed-negative", "linux/amd64-desktop", RenvoCompileOptions{ArenaSize: -1}, false},
		{"managed-tiny", "linux/amd64-desktop", RenvoCompileOptions{ArenaSize: 1}, true},
		{"managed-uncapped", "linux/amd64-desktop", RenvoCompileOptions{}, true},
		{"managed-object", "linux/amd64-desktop", RenvoCompileOptions{ObjectFile: true}, false},
		{"managed-image", "linux/amd64-desktop", RenvoCompileOptions{EmitImage: true}, false},
		{"arena-tiny", "linux/amd64", RenvoCompileOptions{ArenaSize: 1}, false},
		{"arena-large", "linux/amd64", RenvoCompileOptions{ArenaSize: 1073741825}, false},
		{"arena-valid", "linux/amd64", RenvoCompileOptions{ArenaSize: 65536}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary, ok := RenvoCompileSourceToBytesWithOptions(source, tc.target, tc.options)
			if ok != tc.ok || (len(binary) > 0) != tc.ok {
				t.Fatalf("compile = %v, %d bytes; want success %v", ok, len(binary), tc.ok)
			}
		})
	}
}
