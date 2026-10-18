//go:build !browser

package engine

type graphScratch struct{}

func (e *Engine) prepareGraph(c CPU, block *compiledBlock) bool { return false }
