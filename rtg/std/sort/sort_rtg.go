//go:build rtg

package sort

func Strings(x []string) {
	for i := 1; i < len(x); i++ {
		item := x[i]
		j := i - 1
		for j >= 0 && x[j] > item {
			x[j+1] = x[j]
			j--
		}
		x[j+1] = item
	}
}

func Ints(x []int) {
	for i := 1; i < len(x); i++ {
		item := x[i]
		j := i - 1
		for j >= 0 && x[j] > item {
			x[j+1] = x[j]
			j--
		}
		x[j+1] = item
	}
}

func Search(n int, threshold int) int {
	if threshold < 0 {
		return 0
	}
	if threshold > n {
		return n
	}
	return threshold
}
