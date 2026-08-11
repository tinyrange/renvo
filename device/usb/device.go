package usb

const (
	requestGetStatus        = 0
	requestClearFeature     = 1
	requestSetFeature       = 3
	requestSetAddress       = 5
	requestGetDescriptor    = 6
	requestGetConfiguration = 8
	requestSetConfiguration = 9
	requestGetInterface     = 10
	requestSetInterface     = 11
)

// Device owns portable USB protocol state and one controller.
type Device struct {
	controller        Controller
	config            Config
	device            [18]byte
	configuration     []byte
	builder           DescriptorBuilder
	control           [1024]byte
	out               [512]byte
	address           uint8
	pendingAddr       uint8
	pendingAddress    bool
	pendingConfig     uint8
	controlSetup      Setup
	controlFunction   Function
	controlOutPending bool
	controlIn         []byte
	controlInOffset   int
	controlInZLP      bool
	configured        bool
	endpointsOpened   bool
	started           bool
	err               error
}

// New composes config against the controller carried by a board USB port.
func New(port Port, config Config) (*Device, error) {
	if config.VendorID == 0 || config.ProductID == 0 || len(config.Functions) == 0 {
		return nil, ErrInvalidConfig
	}
	d := &Device{controller: port.controller, config: config, builder: newDescriptorBuilder()}
	for _, function := range config.Functions {
		if err := function.Bind(&d.builder); err != nil {
			return nil, err
		}
		function.Attach(d)
	}
	d.configuration = d.builder.finish()
	d.device = deviceDescriptor(config)
	return d, nil
}

// Start configures endpoints and exposes the pull-up to the host.
func (d *Device) Start() error {
	if d.started {
		return nil
	}
	if err := d.controller.Start(); err != nil {
		return err
	}
	d.started = true
	return d.controller.Connect()
}

func (d *Device) openEndpoints() bool {
	if d.endpointsOpened {
		return true
	}
	for index := 0; index < d.builder.endpointCount; index++ {
		if err := d.controller.OpenEndpoint(d.builder.endpoints[index]); err != nil {
			d.err = err
			return false
		}
	}
	d.endpointsOpened = true
	return true
}

// Configured reports whether the host has selected the device configuration.
// It is useful for bounded polling loops and hardware recovery watchdogs.
func (d *Device) Configured() bool { return d.configured }

// Err returns the first asynchronous endpoint-configuration failure.
func (d *Device) Err() error { return d.err }

func min16(a uint16, b int) int {
	if int(a) < b {
		return int(a)
	}
	return b
}

func (d *Device) descriptor(setup Setup) ([]byte, bool) {
	typ, index := byte(setup.Value>>8), byte(setup.Value)
	switch typ {
	case 1:
		return d.device[:min16(setup.Length, len(d.device))], true
	case 2:
		return d.configuration[:min16(setup.Length, len(d.configuration))], true
	case 3:
		if index == 0 {
			d.control[0], d.control[1], d.control[2], d.control[3] = 4, 3, 0x09, 0x04
			return d.control[:min16(setup.Length, 4)], true
		}
		value := ""
		if index == 1 {
			value = d.config.Manufacturer
		} else if index == 2 {
			value = d.config.Product
		} else if index == 3 {
			value = d.config.Serial
		} else {
			return nil, false
		}
		length := stringDescriptor(value, d.control[:])
		return d.control[:min16(setup.Length, length)], true
	case 15:
		for _, function := range d.config.Functions {
			bos := function.BOSDescriptor()
			if len(bos) != 0 {
				return bos[:min16(setup.Length, len(bos))], true
			}
		}
	}
	return nil, false
}

// EndpointSend submits an IN transfer for a bound function.
func (d *Device) EndpointSend(endpoint uint8, data []byte) error {
	return d.controller.Send(endpoint, data)
}

// EndpointReceive arms an OUT transfer for a bound function.
func (d *Device) EndpointReceive(endpoint uint8, buffer []byte) error {
	return d.controller.Receive(endpoint, buffer)
}

// StallEndpoint stalls a bound class endpoint.
func (d *Device) StallEndpoint(endpoint uint8, direction Direction) {
	d.controller.Stall(endpoint, direction)
}

func (d *Device) setup(packet [8]byte) {
	setup := ParseSetup(packet)
	var response []byte
	var owner Function
	ok := false
	if setup.RequestType&0x60 == 0 {
		switch setup.Request {
		case requestGetDescriptor:
			response, ok = d.descriptor(setup)
		case requestSetAddress:
			ok = setup.Value <= 127 && setup.Length == 0
			if ok {
				d.pendingAddr = uint8(setup.Value)
				d.pendingAddress = true
				response = d.control[:0]
			}
		case requestSetConfiguration:
			ok = setup.Value <= 1 && setup.Length == 0
			if ok && setup.Value == 1 {
				ok = d.openEndpoints()
			}
			if ok {
				d.pendingConfig = uint8(setup.Value) + 1
				response = d.control[:0]
			}
		case requestGetConfiguration:
			d.control[0] = 0
			if d.configured {
				d.control[0] = 1
			}
			response, ok = d.control[:min16(setup.Length, 1)], true
		case requestGetStatus:
			d.control[0], d.control[1] = 0, 0
			response, ok = d.control[:min16(setup.Length, 2)], true
		case requestClearFeature, requestSetFeature:
			response, ok = d.control[:0], setup.Length == 0
		}
	}
	if !ok {
		for _, function := range d.config.Functions {
			length := function.Control(setup, d.control[:])
			if length >= 0 && length <= len(d.control) {
				response = d.control[:min16(setup.Length, length)]
				ok = true
				owner = function
				break
			}
		}
	}
	if !ok && setup.RequestType&0x60 == 0 {
		if setup.Request == requestGetInterface {
			d.control[0] = 0
			response, ok = d.control[:min16(setup.Length, 1)], true
		} else if setup.Request == requestSetInterface && setup.Value == 0 && setup.Length == 0 {
			response, ok = d.control[:0], true
		}
	}
	if !ok {
		d.controller.Stall(0, Direction(setup.RequestType>>7))
		return
	}
	if setup.RequestType&0x80 == 0 && setup.Length != 0 {
		if owner == nil || int(setup.Length) > len(d.out) {
			d.controller.Stall(0, Out)
			return
		}
		d.controlSetup = setup
		d.controlFunction = owner
		d.controlOutPending = true
		_ = d.controller.Receive(0, d.out[:setup.Length])
		return
	}
	if setup.RequestType&0x80 != 0 {
		d.controlIn = response
		d.controlInOffset = 0
		d.controlInZLP = len(response) != 0 && len(response)%64 == 0 && len(response) < int(setup.Length)
		d.sendControlIn()
		_ = d.controller.Receive(0, d.out[:0])
	} else {
		_ = d.controller.Send(0, response)
	}
}

func (d *Device) sendControlIn() {
	if d.controlInOffset < len(d.controlIn) {
		end := d.controlInOffset + 64
		if end > len(d.controlIn) {
			end = len(d.controlIn)
		}
		_ = d.controller.Send(0, d.controlIn[d.controlInOffset:end])
		d.controlInOffset = end
		return
	}
	if d.controlInZLP {
		d.controlInZLP = false
		_ = d.controller.Send(0, d.control[:0])
		return
	}
	d.controlIn = nil
}

// Poll processes a bounded number of controller events without a scheduler.
func (d *Device) Poll() {
	for count := 0; count < 16; count++ {
		var event Event
		if !d.controller.Poll(&event) {
			return
		}
		switch event.Kind {
		case EventReset:
			d.address, d.pendingAddr, d.configured = 0, 0, false
			d.pendingAddress, d.controlOutPending = false, false
			d.pendingConfig = 0
			d.controlIn = nil
		case EventSetup:
			d.setup(event.Setup)
		case EventOut:
			if event.Endpoint == 0 && d.controlOutPending {
				accepted := d.controlFunction.ControlOut(d.controlSetup, event.Data)
				d.controlOutPending = false
				d.controlFunction = nil
				if accepted {
					_ = d.controller.Send(0, d.control[:0])
				} else {
					d.controller.Stall(0, In)
				}
			}
		case EventInComplete:
			if event.Endpoint == 0 {
				if d.controlIn != nil {
					d.sendControlIn()
				} else if d.pendingAddress {
					d.address = d.pendingAddr
					d.pendingAddr = 0
					d.pendingAddress = false
					d.controller.SetAddress(d.address)
				} else if d.pendingConfig != 0 {
					d.configured = d.pendingConfig == 2
					d.pendingConfig = 0
					for _, function := range d.config.Functions {
						function.Configured(d.configured)
					}
				}
			}
		}
		for _, function := range d.config.Functions {
			function.Handle(event)
		}
	}
}
