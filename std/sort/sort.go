package sort

func Strings(x []string) {
	for i := 1; i < len(x); i++ {
		item := x[i]
		j := i - 1
		for j >= 0 && stringGreater(x[j], item) {
			x[j+1] = x[j]
			j--
		}
		x[j+1] = item
	}
}

func stringGreater(left string, right string) bool {
	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	for i := 0; i < length; i++ {
		if left[i] > right[i] {
			return true
		}
		if left[i] < right[i] {
			return false
		}
	}
	return len(left) > len(right)
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

func Search(n int, f func(int) bool) int {
	i := 0
	j := n
	for i < j {
		h := int(uint(i+j) >> 1)
		if !f(h) {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}
