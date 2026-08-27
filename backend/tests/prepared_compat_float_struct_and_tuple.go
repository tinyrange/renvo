package main

type preparedCompatPoint struct {
	X float64
	Y float64
}

type preparedCompatReading struct {
	First  uint16
	Second uint16
}

type preparedCompatState struct {
	Scale float64
}

type preparedCompatDevice struct{}

func (*preparedCompatDevice) Read() (preparedCompatReading, error) {
	return preparedCompatReading{First: 42, Second: 7}, nil
}

func (d *preparedCompatDevice) Measure() (uint16, error) {
	reading, err := d.Read()
	return reading.First, err
}

func appMain() int {
	state := preparedCompatState{}
	state.Scale = 1.0
	point := preparedCompatPoint{X: 1.5, Y: 2.5}
	device := preparedCompatDevice{}
	value, err := device.Measure()
	if state.Scale != 1.0 || point.X != 1.5 || point.Y != 2.5 || value != 42 || err != nil {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
