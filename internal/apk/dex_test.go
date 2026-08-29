package apk

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestBuildDEXProducesVerifiedV2APK(t *testing.T) {
	dex := testDEX()
	config, err := ParseConfig([]byte("package=dev.renvo.dextest\n" +
		"name=Renvo DEX Test\nversion_code=3\nversion_name=1.2\n" +
		"min_sdk=24\ntarget_sdk=35\n"))
	if err != nil {
		t.Fatal(err)
	}
	image, err := BuildDEX(dex, config)
	if err != nil {
		t.Fatal(err)
	}
	verifyV2Signature(t, image)
	reader, err := zip.NewReader(bytes.NewReader(image), int64(len(image)))
	if err != nil {
		t.Fatal(err)
	}
	entries := make(map[string][]byte)
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = data
	}
	if len(entries) != 2 || !bytes.Equal(entries["classes.dex"], dex) {
		t.Fatalf("DEX APK entries are incomplete: %v", entries)
	}
	manifest := entries["AndroidManifest.xml"]
	for _, required := range [][]byte{
		[]byte("dev.renvo.dextest"),
		[]byte("dev.renvo.app.RenvoActivity"),
	} {
		if !bytes.Contains(manifest, required) {
			t.Errorf("DEX manifest omits %q", required)
		}
	}
	if bytes.Contains(manifest, []byte("android.app.NativeActivity")) ||
		bytes.Contains(manifest, []byte("android.app.lib_name")) {
		t.Fatal("DEX manifest retained NativeActivity metadata")
	}
}

func TestBuildDEXRejectsNonDEXPayload(t *testing.T) {
	config := DefaultConfig()
	config.Package = "dev.renvo.invalid"
	if _, err := BuildDEX([]byte("not dex"), config); err == nil {
		t.Fatal("BuildDEX accepted a non-DEX payload")
	}
}

func testDEX() []byte {
	image := make([]byte, 112)
	copy(image, []byte("dex\n035\x00"))
	binary.LittleEndian.PutUint32(image[32:36], uint32(len(image)))
	binary.LittleEndian.PutUint32(image[36:40], 112)
	binary.LittleEndian.PutUint32(image[40:44], 0x12345678)
	return image
}
