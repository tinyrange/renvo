package main

import "fmt"

// runtimeNumberDefault selects the registry's compatibility role, not a public
// OS/ISA spelling. It does not select the CLI default or a fixed target's ABI.
// Validate the role before any generator writes, and also at each projection
// entrypoint so callers cannot silently fall back to a familiar target name.
func runtimeNumberDefault(descriptors []sourceDescriptor) (*sourceDescriptor, error) {
	var selected *sourceDescriptor
	for i := range descriptors {
		descriptor := &descriptors[i]
		if !descriptor.RuntimeNumberDefault {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("multiple runtime number defaults: %q and %q", selected.Name, descriptor.Name)
		}
		if descriptor.Constant == "" || descriptor.BackendID < 1 || descriptor.OSID < 1 || descriptor.ISAID < 1 {
			return nil, fmt.Errorf("runtime number default %q requires a backend projection", descriptor.Name)
		}
		for _, operation := range []string{"read", "write", "read_at", "write_at", "open", "close", "chmod", "exit"} {
			if _, ok := descriptor.RuntimeNumbers[operation]; !ok {
				return nil, fmt.Errorf("runtime number default %q has no %s runtime number", descriptor.Name, operation)
			}
		}
		selected = descriptor
	}
	if selected == nil {
		return nil, fmt.Errorf("target registry requires one runtime_number_default")
	}
	return selected, nil
}
