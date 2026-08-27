package second

type Device struct{}

func (*Device) Read() int { return 7 }
