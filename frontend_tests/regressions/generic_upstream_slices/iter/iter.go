package iter
type Seq[V any] func(yield func(V)bool)
type Seq2[K,V any] func(yield func(K,V)bool)
