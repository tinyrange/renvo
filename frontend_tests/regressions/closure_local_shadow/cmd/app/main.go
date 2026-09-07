package main

type Callback func() int

func main() {
	err := 10
	callback := Callback(func() int {
		total := err
		if err := err + 1; err != 11 { panic("initializer scope") } else { total += err }
		total += err
		{
			var err = err + 2
			total += err
		}
		total += err
		return total
	})
	if callback() != 53 || err != 10 { panic("shadow capture") }
	println("PASS")
}
