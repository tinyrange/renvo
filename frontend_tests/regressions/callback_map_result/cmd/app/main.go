package main

type Loader func(string) (map[string]int, error)
type Module struct{ loader Loader }
type Compiler struct{ module *Module }

func load(name string) (map[string]int, error) {
	return map[string]int{name: 42}, nil
}

func (c *Compiler) run() {
	values, err := c.module.loader("answer")
	if err != nil {
		panic("loader")
	}
	value, ok := values["answer"]
	if !ok || value != 42 {
		panic("present")
	}
	value, ok = values["missing"]
	if ok || value != 0 {
		panic("absent")
	}
}

func main() {
	c := &Compiler{module: &Module{loader: load}}
	c.run()
	println("PASS")
}
