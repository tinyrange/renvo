package check

// These functions copy compact checker results into the high arena in a
// self-hosted frontend, allowing the low arena's per-function scratch to be
// reset immediately. Host-built frontends use the identity implementations.
func renvo_runtime_ArenaPersistCheckNameRefs(value []CoreNameRef) []CoreNameRef { return value }

func renvo_runtime_ArenaPersistCheckSelectorRefs(value []CoreSelectorRef) []CoreSelectorRef {
	return value
}

func renvo_runtime_ArenaPersistCheckTypeRefs(value []CoreTypeRef) []CoreTypeRef { return value }

func renvo_runtime_ArenaPersistCheckBools(value []bool) []bool { return value }
