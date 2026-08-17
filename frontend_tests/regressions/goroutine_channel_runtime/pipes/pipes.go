// Package pipes provides named and directional channel values used by the
// cross-package concurrency regression.
package pipes

type Values chan int

func New() Values { return make(Values, 1) }

func Send(values chan<- int, value int) { values <- value }

func Receive(values <-chan int) int { return <-values }
