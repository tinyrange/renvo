package rtg

// Managed ABI 1 is a compiler/runtime boundary, not an OS or ISA identity.
// Runtime source supplies Init, Request and CollectBoundary. The emitters own
// calling conventions, root snapshots and startup/global-address materialization.
func validateManagedTarget(document Document, target ResolvedTarget) (Diagnostic, bool) {
	abi, ok := integerField(document, target.Runtime, "managed_heap_abi")
	bits, widthOK := integerField(document, target.Runtime, "managed_word_bits")
	if !ok || abi != 1 || !widthOK || bits != target.Descriptor.WordBits || bits != target.Descriptor.PointerBits || target.Descriptor.Executable == "" {
		return resolveDiagnostic(document, target.Declaration, "RTG-RESOLVE-023", "desktop memory requires managed_heap_abi = 1, matching managed_word_bits, and executable output"), false
	}
	for _, field := range []string{"managed_source", "managed_startup", "managed_boundary"} {
		if _, ok := architectureGoHook(target.Runtime, field); !ok {
			return resolveDiagnostic(document, target.Runtime, "RTG-RESOLVE-023", "desktop memory requires runtime binding "+field), false
		}
	}
	return Diagnostic{}, true
}

func managedHookName(document Document, runtime Declaration, field string) string {
	hook, _ := architectureGoHook(runtime, field)
	return "rtg" + exportedName(document.Unit) + exportedName(hook)
}

func appendPreparedManagedHeap(source []byte, document Document, target ResolvedTarget) []byte {
	active := target.Descriptor.MemoryStrategy == "desktop"
	source = append(source, "\nfunc renvoManagedTarget(target int) bool { return "...)
	if active {
		source = append(source, "true"...)
	} else {
		source = append(source, "false"...)
	}
	source = append(source, " }\nfunc renvoManagedRuntimeSource(target int) string {\n"...)
	if active {
		source = append(source, "if renvoManagedTarget(target) { return "+managedHookName(document, target.Runtime, "managed_source")+"() }\n"...)
	}
	source = append(source, "return \"\"\n}\nfunc renvoManagedStartup(target int, out *renvoAsm, limit int, label int) {\n"...)
	if active {
		source = append(source, "if renvoManagedTarget(target) { "+managedHookName(document, target.Runtime, "managed_startup")+"(out,limit,label); return }\n"...)
	}
	source = append(source, "out.patchFailed = true\n}\nfunc renvoManagedBoundary(target int, out *renvoAsm, label int) {\n"...)
	if active {
		source = append(source, "if renvoManagedTarget(target) { "+managedHookName(document, target.Runtime, "managed_boundary")+"(out,label); return }\n"...)
	}
	return append(source, "out.patchFailed = true\n}\n"...)
}

// Project definition-owned hooks into the bundled compiler kernel.
func appendBundledManagedHeap(source []byte, definitions []ResolveResult) GenerateResult {
	var targets []ResolvedTarget
	var documents []Document
	emitted := make(map[string]string)
	for _, resolved := range definitions {
		if !resolved.Ok {
			return GenerateResult{Diagnostics: resolved.Diagnostics}
		}
		var roots []string
		for _, target := range resolved.Targets {
			if target.Descriptor.MemoryStrategy != "desktop" {
				continue
			}
			targets = append(targets, target)
			documents = append(documents, resolved.Document)
			for _, field := range []string{"managed_source", "managed_startup", "managed_boundary"} {
				name, _ := architectureGoHook(target.Runtime, field)
				roots = append(roots, name)
			}
		}
		if len(roots) == 0 {
			continue
		}
		var excluded []string
		for _, part := range reachableEmbeddedGoParts(resolved.Document, roots, nil) {
			key := resolved.Document.Unit + "/" + part.name
			if old, exists := emitted[key]; exists {
				if old != string(part.source) {
					return GenerateResult{Diagnostics: []Diagnostic{resolveDiagnostic(resolved.Document, Declaration{}, "RTG-GENERATE-020", "conflicting managed helper "+key)}}
				}
				excluded = append(excluded, part.name)
			} else {
				emitted[key] = string(part.source)
			}
		}
		source = appendReachableEmbeddedGoExcluding(source, resolved.Document, roots, true, nil, excluded)
	}
	source = append(source, "\nfunc renvoManagedTarget(target int) bool {\n"...)
	for _, target := range targets {
		source = append(source, "if target == renvoParseTargetArg("...)
		source = appendQuoted(source, target.Descriptor.Name)
		source = append(source, ") { return true }\n"...)
	}
	source = append(source, "return false\n}\nfunc renvoManagedRuntimeSource(target int) string {\n"...)
	for i, target := range targets {
		source = append(source, "if target == renvoParseTargetArg("...)
		source = appendQuoted(source, target.Descriptor.Name)
		source = append(source, ") { return "+managedHookName(documents[i], target.Runtime, "managed_source")+"() }\n"...)
	}
	source = append(source, "return \"\"\n}\nfunc renvoManagedStartup(target int, out *renvoAsm, limit int, label int) {\n"...)
	for i, target := range targets {
		source = append(source, "if target == renvoParseTargetArg("...)
		source = appendQuoted(source, target.Descriptor.Name)
		source = append(source, ") { "+managedHookName(documents[i], target.Runtime, "managed_startup")+"(out,limit,label); return }\n"...)
	}
	source = append(source, "out.patchFailed = true\n}\nfunc renvoManagedBoundary(target int, out *renvoAsm, label int) {\n"...)
	for i, target := range targets {
		source = append(source, "if target == renvoParseTargetArg("...)
		source = appendQuoted(source, target.Descriptor.Name)
		source = append(source, ") { "+managedHookName(documents[i], target.Runtime, "managed_boundary")+"(out,label); return }\n"...)
	}
	source = append(source, "out.patchFailed = true\n}\n"...)
	return GenerateResult{Source: source, Ok: true}
}
