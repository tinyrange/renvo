package rtg

// TargetMetrics is a non-enforcing review aid. It reports how much authored
// declarative structure and embedded Go is retained by one resolved target.
// No validity or performance decision depends on these values.
type TargetMetrics struct {
	DeclarativeBytes int
	ReachableGoBytes int
	CatalogGoBytes   int
	ReachableGoDecls int
	CatalogGoDecls   int
}

func MeasureTarget(document Document, target ResolvedTarget) TargetMetrics {
	var metrics TargetMetrics
	declarations := targetSemanticDeclarations(document, target)
	for i := 0; i < len(declarations); i++ {
		metrics.DeclarativeBytes += len(declarations[i].body)
	}
	parts := reachableEmbeddedGoParts(document, targetGoRoots(document, target), nil)
	for i := 0; i < len(parts); i++ {
		metrics.ReachableGoBytes += len(parts[i].source)
	}
	metrics.ReachableGoDecls = len(parts)
	all := reachableEmbeddedGoParts(document, embeddedGoNames(document), nil)
	for i := 0; i < len(all); i++ {
		metrics.CatalogGoBytes += len(all[i].source)
	}
	metrics.CatalogGoDecls = len(all)
	return metrics
}
