package driver

import (
	"bytes"
	"testing"
)

func TestPackageBrowserHTMLIsSelfContained(t *testing.T) {
	html := PackageBrowserHTML([]byte{0, 'a', 's', 'm'})
	for _, fragment := range [][]byte{
		[]byte("<!doctype html>"),
		[]byte("AGFzbQ=="),
		[]byte("webgl2"),
		[]byte("renvo_browser_step"),
		[]byte("id=\"desktop\""),
		[]byte("createElement(\"canvas\")"),
		[]byte("localStorage"),
		[]byte("launchFile"),
		[]byte(".renvo-window.floating"),
		[]byte("ResizeObserver"),
		[]byte("/workspace"),
		[]byte("setWindowTimer"),
	} {
		if !bytes.Contains(html, fragment) {
			t.Fatalf("browser HTML is missing %q", string(fragment))
		}
	}
	if bytes.Contains(html, []byte("<script src=")) || bytes.Contains(html, []byte("<link rel=")) {
		t.Fatal("browser HTML contains an external dependency")
	}
}

func TestAppendBase64(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{nil, ""},
		{[]byte("f"), "Zg=="},
		{[]byte("fo"), "Zm8="},
		{[]byte("foo"), "Zm9v"},
		{[]byte("hello"), "aGVsbG8="},
	}
	for _, test := range tests {
		if got := string(appendBase64(nil, test.data)); got != test.want {
			t.Fatalf("base64(%q) = %q, want %q", string(test.data), got, test.want)
		}
	}
}
