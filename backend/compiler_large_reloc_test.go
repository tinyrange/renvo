package main

import (
	"strings"
	"testing"
)

func TestLargeGlobalByteSliceGrowsAbsoluteRelocations(t *testing.T) {
	oldFixedTarget := renvoFixedTarget
	t.Cleanup(func() { renvoFixedTarget = oldFixedTarget })
	renvoFixedTarget = renvoTargetDarwinArm64

	var source strings.Builder
	source.WriteString("package main\nvar data = []byte{")
	for i := 0; i < 16385; i++ {
		source.WriteString("0,")
	}
	source.WriteString("}\nfunc appMain() int { if len(data) == 16385 { print(\"PASS\\n\") }; return 0 }\n")

	image, ok := RenvoCompileSourceToBytesStrip([]byte(source.String()), "darwin/arm64", true)
	if !ok {
		t.Fatal("large Darwin byte-slice literal failed to compile")
	}
	if len(image) < 4 || string(image[:4]) != "\xcf\xfa\xed\xfe" {
		t.Fatalf("large Darwin byte-slice image prefix = %x", image[:min(len(image), 4)])
	}
}
