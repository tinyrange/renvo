package main

import "testing"

func TestValidBuildTag(t *testing.T) {
	for _, tag := range []string{"renvo_prepared", "linux", "go1.25", "386"} {
		if !validBuildTag(tag) {
			t.Errorf("validBuildTag(%q) = false", tag)
		}
	}
	for _, tag := range []string{"", "linux && amd64", "tag-name", "tag\nother"} {
		if validBuildTag(tag) {
			t.Errorf("validBuildTag(%q) = true", tag)
		}
	}
}
