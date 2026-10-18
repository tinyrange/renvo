package targetinfo

import "testing"

func TestLookupProjectionsMatchDescriptors(t *testing.T) {
	all := All()
	tags := []string{"", "unknown", "amd", "unixx"}
	capabilities := []string{"", "unknown", "host", "hostedx"}
	for _, d := range all {
		tags = append(tags, d.Tags...)
		capabilities = append(capabilities, d.Capabilities...)
	}
	for _, d := range all {
		names := append([]string{d.Name}, d.Aliases...)
		for _, name := range names {
			got, ok := Lookup(name)
			if !ok || got.Name != d.Name {
				t.Fatalf("lookup %q: %#v %v", name, got, ok)
			}
			bound, fingerprint, version, ok := Binding(name)
			if !ok || bound != d.Name || fingerprint != string(d.Definition[:]) || version != d.DescriptorVersion {
				t.Fatalf("binding %q differs", name)
			}
			for _, capability := range capabilities {
				if HasCapability(name, capability) != stringSliceContains(d.Capabilities, capability) {
					t.Errorf("capability %q/%q differs", name, capability)
				}
			}
		}
		if IsAdvertised(d.Name) != d.Advertised || DefaultArena(d.Name) != d.DefaultArena {
			t.Errorf("policy %q differs", d.Name)
		}
		for _, tag := range tags {
			if HasBuildTag(d.Name, tag) != stringSliceContains(d.Tags, tag) {
				t.Errorf("tag %q/%q differs", d.Name, tag)
			}
		}
	}
	for _, name := range []string{"", "unknown", "linux", "linux/amd64-extra"} {
		if _, ok := Lookup(name); ok {
			t.Errorf("unknown lookup %q", name)
		}
		if _, _, _, ok := Binding(name); ok {
			t.Errorf("unknown binding %q", name)
		}
		if IsAdvertised(name) || HasBuildTag(name, "linux") || HasCapability(name, "heap") || DefaultArena(name) != 0 || MemoryStrategy(name) != "" {
			t.Errorf("unknown policy %q", name)
		}
	}
}
