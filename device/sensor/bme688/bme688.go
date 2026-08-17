// Package bme688 drives a Bosch BME688 environmental sensor over I2C.
//
// The sensor directly provides calibrated temperature, pressure, humidity,
// and gas resistance. Bosch's separate BSEC software is required to derive
// IAQ, equivalent CO2, or breath-VOC values from these measurements.
package bme688

const (
	// AddressLow is selected when the BME688 SDO pin is grounded.
	AddressLow = uint16(0x76)
	// AddressHigh is selected when the BME688 SDO pin is tied high. The
	// M5Stack ENV Pro Unit uses this address.
	AddressHigh = uint16(0x77)

	chipIDRegister  = byte(0xd0)
	softReset       = byte(0xe0)
	variantRegister = byte(0xf0)
	coeff1Register  = byte(0x8a)
	coeff2Register  = byte(0xe1)
	coeff3Register  = byte(0x00)
	field0Register  = byte(0x1d)
	resHeatRegister = byte(0x5a)
	gasWaitRegister = byte(0x64)
	ctrlGas0        = byte(0x70)
	ctrlGas1        = byte(0x71)
	ctrlHumidity    = byte(0x72)
	ctrlMeasurement = byte(0x74)

	chipID           = byte(0x61)
	resetCommand     = byte(0xb6)
	variantGasHigh   = byte(0x01)
	forcedMode       = byte(0x01)
	newDataMask      = byte(0x80)
	gasValidMask     = byte(0x20)
	heaterStableMask = byte(0x10)
)

// Bus is the minimal I2C and timing capability required by the sensor.
type Bus interface {
	Tx(address uint16, write, read []byte) error
	DelayMilliseconds(uint32)
}

// Reading is one calibrated environmental measurement. Temperature is in
// hundredths of a degree Celsius and Humidity is in thousandths of percent
// relative humidity, preserving the precision of Bosch's integer reference
// compensation. Pressure is in pascals and GasResistance is in ohms.
type Reading struct {
	Temperature   int16
	Pressure      uint32
	Humidity      uint32
	GasResistance uint32
	GasValid      bool
	HeaterStable  bool
}

// Device is one BME688 attached to a bus.
type Device struct {
	bus     Bus
	address uint16
	variant byte
	calib   calibration
}

type calibration struct {
	h1, h2                         uint16
	h3, h4, h5, h7                 int8
	h6                             uint8
	gh1, gh3                       int8
	gh2                            int16
	t1                             uint16
	t2                             int16
	t3                             int8
	p1                             uint16
	p2, p4, p5, p8, p9             int16
	p3, p6, p7                     int8
	p10                            uint8
	tFine                          int32
	resHeatRange                   uint8
	resHeatValue, rangeSwitchError int8
}

// New binds a BME688 at the selected seven-bit I2C address to bus.
func New(bus Bus, address uint16) *Device {
	return &Device{bus: bus, address: address}
}

func (d *Device) read(register byte, data []byte) error {
	command := [1]byte{register}
	return d.bus.Tx(d.address, command[:], data)
}

func (d *Device) write(register, value byte) error {
	command := [2]byte{register, value}
	return d.bus.Tx(d.address, command[:], nil)
}

// Initialize resets the sensor, verifies its chip ID, loads its factory
// calibration, and configures the Bosch reference forced-mode profile:
// temperature 2x, pressure 1x, humidity 16x, and a 300 C heater for 100 ms.
func (d *Device) Initialize() error {
	if err := d.write(softReset, resetCommand); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(10)

	identity := [1]byte{}
	if err := d.read(chipIDRegister, identity[:]); err != nil {
		return err
	}
	if identity[0] != chipID {
		return ErrDeviceID
	}
	if err := d.read(variantRegister, identity[:]); err != nil {
		return err
	}
	d.variant = identity[0]

	coefficients := [42]byte{}
	if err := d.read(coeff1Register, coefficients[0:23]); err != nil {
		return err
	}
	if err := d.read(coeff2Register, coefficients[23:37]); err != nil {
		return err
	}
	if err := d.read(coeff3Register, coefficients[37:42]); err != nil {
		return err
	}
	d.calib.decode(&coefficients)

	// Humidity must be written before ctrlMeasurement for the setting to latch.
	if err := d.write(ctrlHumidity, 5); err != nil { // 16x humidity
		return err
	}
	if err := d.write(ctrlMeasurement, 2<<5|1<<2); err != nil { // 2x temperature, 1x pressure, sleep
		return err
	}
	if err := d.write(resHeatRegister, d.calib.heaterResistance(300)); err != nil {
		return err
	}
	if err := d.write(gasWaitRegister, encodeGasWait(100)); err != nil {
		return err
	}

	gasControl := [2]byte{}
	if err := d.read(ctrlGas0, gasControl[:]); err != nil {
		return err
	}
	gasControl[0] &^= 0x08 // heater enabled
	gasControl[1] &^= 0x3f
	if d.variant == variantGasHigh {
		gasControl[1] |= 0x20
	} else {
		gasControl[1] |= 0x10
	}
	if err := d.write(ctrlGas0, gasControl[0]); err != nil {
		return err
	}
	return d.write(ctrlGas1, gasControl[1])
}

// Read starts a forced measurement and returns its compensated result. A read
// takes approximately 143 ms with the profile installed by Initialize.
func (d *Device) Read() (Reading, error) {
	var result Reading
	err := d.ReadInto(&result)
	return result, err
}

// ReadInto stores one forced measurement in caller-provided storage.
func (d *Device) ReadInto(result *Reading) error {
	control := [1]byte{}
	if err := d.read(ctrlMeasurement, control[:]); err != nil {
		return err
	}
	if err := d.write(ctrlMeasurement, control[0]&^0x03|forcedMode); err != nil {
		return err
	}
	d.bus.DelayMilliseconds(143)

	field := [17]byte{}
	for attempt := 0; attempt < 5; attempt++ {
		if err := d.read(field0Register, field[:]); err != nil {
			return err
		}
		if field[0]&newDataMask != 0 {
			*result = d.compensate(&field)
			return nil
		}
		d.bus.DelayMilliseconds(10)
	}
	return ErrNoNewData
}

func unsigned16(msb, lsb byte) uint16 { return uint16(msb)<<8 | uint16(lsb) }
func signed16(msb, lsb byte) int16    { return int16(unsigned16(msb, lsb)) }

func (c *calibration) decode(data *[42]byte) {
	c.t2 = signed16(data[1], data[0])
	c.t3 = int8(data[2])
	c.p1 = unsigned16(data[5], data[4])
	c.p2 = signed16(data[7], data[6])
	c.p3 = int8(data[8])
	c.p4 = signed16(data[11], data[10])
	c.p5 = signed16(data[13], data[12])
	c.p7 = int8(data[14])
	c.p6 = int8(data[15])
	c.p8 = signed16(data[19], data[18])
	c.p9 = signed16(data[21], data[20])
	c.p10 = data[22]
	c.h2 = uint16(data[23])<<4 | uint16(data[24])>>4
	c.h1 = uint16(data[25])<<4 | uint16(data[24]&0x0f)
	c.h3 = int8(data[26])
	c.h4 = int8(data[27])
	c.h5 = int8(data[28])
	c.h6 = data[29]
	c.h7 = int8(data[30])
	c.t1 = unsigned16(data[32], data[31])
	c.gh2 = signed16(data[34], data[33])
	c.gh1 = int8(data[35])
	c.gh3 = int8(data[36])
	c.resHeatValue = int8(data[37])
	c.resHeatRange = (data[39] & 0x30) / 16
	c.rangeSwitchError = int8(data[41]&0xf0) / 16
}

func encodeGasWait(duration uint16) byte {
	if duration >= 0xfc0 {
		return 0xff
	}
	factor := byte(0)
	for duration > 0x3f {
		duration /= 4
		factor++
	}
	return byte(duration) + factor*64
}

func (c *calibration) heaterResistance(temperature uint16) byte {
	if temperature > 400 {
		temperature = 400
	}
	// The reference calculation assumes a 25 C ambient temperature.
	var1 := (int32(25) * int32(c.gh3) / 1000) * 256
	var2 := (int32(c.gh1) + 784) * (((int32(c.gh2)+154009)*int32(temperature)*5/100 + 3276800) / 10)
	var3 := var1 + var2/2
	var4 := var3 / (int32(c.resHeatRange) + 4)
	var5 := 131*int32(c.resHeatValue) + 65536
	valueX100 := (var4/var5 - 250) * 34
	return byte((valueX100 + 50) / 100)
}

func (d *Device) compensate(field *[17]byte) Reading {
	pressureADC := uint32(field[2])<<12 | uint32(field[3])<<4 | uint32(field[4])>>4
	temperatureADC := uint32(field[5])<<12 | uint32(field[6])<<4 | uint32(field[7])>>4
	humidityADC := uint16(field[8])<<8 | uint16(field[9])
	gasLowADC := uint16(field[13])<<2 | uint16(field[14])>>6
	gasHighADC := uint16(field[15])<<2 | uint16(field[16])>>6

	temperature := d.calib.temperature(temperatureADC)
	pressure := d.calib.pressure(pressureADC)
	humidity := d.calib.humidity(humidityADC)
	status := field[14]
	gas := d.calib.gasResistanceLow(gasLowADC, field[14]&0x0f)
	if d.variant == variantGasHigh {
		status = field[16]
		gas = gasResistanceHigh(gasHighADC, field[16]&0x0f)
	}
	return Reading{
		Temperature:   temperature,
		Pressure:      pressure,
		Humidity:      humidity,
		GasResistance: gas,
		GasValid:      status&gasValidMask != 0,
		HeaterStable:  status&heaterStableMask != 0,
	}
}

// The compensation below is the integer algorithm from Bosch Sensortec's
// BSD-3-Clause BME68x Sensor API v4.4.8.
func (c *calibration) temperature(adc uint32) int16 {
	var1 := (int32(adc) >> 3) - (int32(c.t1) << 1)
	var2 := var1 * int32(c.t2) >> 11
	var3 := ((var1 >> 1) * (var1 >> 1)) >> 12
	var3 = var3 * (int32(c.t3) << 4) >> 14
	c.tFine = var2 + var3
	return int16((c.tFine*5 + 128) >> 8)
}

func (c *calibration) pressure(adc uint32) uint32 {
	var1 := (c.tFine >> 1) - 64000
	var2 := ((((var1 >> 2) * (var1 >> 2)) >> 11) * int32(c.p6)) >> 2
	var2 += (var1 * int32(c.p5)) << 1
	var2 = (var2 >> 2) + (int32(c.p4) << 16)
	var1 = (((((var1>>2)*(var1>>2))>>13)*(int32(c.p3)<<5))>>3 + (int32(c.p2)*var1)>>1) >> 18
	var1 = ((32768 + var1) * int32(c.p1)) >> 15
	pressure := int32(1048576 - adc)
	pressure = (pressure - (var2 >> 12)) * 3125
	if pressure >= 0x40000000 {
		pressure = pressure / var1 << 1
	} else {
		pressure = (pressure << 1) / var1
	}
	var1 = int32(c.p9) * (((pressure >> 3) * (pressure >> 3)) >> 13) >> 12
	var2 = (pressure >> 2) * int32(c.p8) >> 13
	var3 := (pressure >> 8) * (pressure >> 8) * (pressure >> 8) * int32(c.p10) >> 17
	pressure += (var1 + var2 + var3 + (int32(c.p7) << 7)) >> 4
	return uint32(pressure)
}

func (c *calibration) humidity(adc uint16) uint32 {
	temperature := (c.tFine*5 + 128) >> 8
	var1 := int32(adc) - int32(c.h1)*16 - ((temperature * int32(c.h3) / 100) >> 1)
	var2 := int32(c.h2) * (temperature*int32(c.h4)/100 + ((temperature * (temperature * int32(c.h5) / 100) >> 6) / 100) + 1<<14) >> 10
	var3 := var1 * var2
	var4 := ((int32(c.h6) << 7) + temperature*int32(c.h7)/100) >> 4
	var5 := ((var3 >> 14) * (var3 >> 14)) >> 10
	var6 := var4 * var5 >> 1
	result := ((var3 + var6) >> 10) * 1000 >> 12
	if result > 100000 {
		result = 100000
	} else if result < 0 {
		result = 0
	}
	return uint32(result)
}

func (c *calibration) gasResistanceLow(adc uint16, gasRange byte) uint32 {
	lookup1 := [16]uint32{2147483647, 2147483647, 2147483647, 2147483647, 2147483647, 2126008810, 2147483647, 2130303777, 2147483647, 2147483647, 2143188679, 2136746228, 2147483647, 2126008810, 2147483647, 2147483647}
	lookup2 := [16]uint32{4096000000, 2048000000, 1024000000, 512000000, 255744255, 127110228, 64000000, 32258064, 16016016, 8000000, 4000000, 2000000, 1000000, 500000, 250000, 125000}
	index := int(gasRange)
	var1 := (int64(1340+5*int64(c.rangeSwitchError)) * int64(lookup1[index])) >> 16
	var2 := (int64(adc)<<15 - 16777216) + var1
	var3 := int64(lookup2[index]) * var1 >> 9
	return uint32((var3 + (var2 >> 1)) / var2)
}

func gasResistanceHigh(adc uint16, gasRange byte) uint32 {
	var1 := uint32(262144) >> gasRange
	var2 := (int32(adc)-512)*3 + 4096
	return (10000 * var1 / uint32(var2)) * 100
}

type sensorError string

func (e sensorError) Error() string { return string(e) }

const (
	// ErrDeviceID reports that the device at the selected address is not a BME688.
	ErrDeviceID sensorError = "bme688 device id mismatch"
	// ErrNoNewData reports that a forced measurement did not complete after the
	// documented measurement time and five additional polls.
	ErrNoNewData sensorError = "bme688 measurement produced no new data"
)
