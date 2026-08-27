package first

type Reading struct {
	First  uint16
	Second uint16
}

type Device struct{}

func (*Device) Read() (Reading, error) {
	return Reading{First: 42, Second: 7}, nil
}

func (d *Device) Measure() (uint16, error) {
	reading, err := d.Read()
	return reading.First, err
}
