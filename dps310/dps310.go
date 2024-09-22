package dps310

// Datasheet:
// https://www.infineon.com/dgdl/Infineon-DPS310-DataSheet-v01_02-EN.pdf?fileId=5546d462576f34750157750826c42242

import (
	"fmt"
	"machine"
	"math"
	"time"
)

// hardware interfacing with a DPS310
type Config struct {
	SPI        *machine.SPI
	CSD        machine.Pin
	SPI_Config machine.SPIConfig
}

type DPS310 struct {
	c0, c1, c01, c00, c10, c11, c20, c21, c30 float32
	tempScale, pressureScale                  float32
	tempRate, pressureRate                    Rate
	tempOs, pressureOs                        Precision
	seaLevelPressure                          float32
	mode                                      Mode
	rwBuf                                     [4]byte
	spi                                       *machine.SPI
	cs                                        machine.Pin
}

func NewSPI(c Config) (*DPS310, error) {
	d := &DPS310{
		spi: c.SPI,
		cs:  c.CSD,
	}

	err := d.spi.Configure(c.SPI_Config)

	if err != nil {
		return nil, err
	}

	d.cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.cs.Low()
	time.Sleep(CSB_Setup_Time * time.Nanosecond)
	d.cs.High()
	time.Sleep(CSB_Setup_Time * time.Nanosecond)

	return d, nil
}

// initialization code for sensor on  SPI
func (d *DPS310) Init() (err error) {

	chipId, err := d.readRegister(Product_ID)
	if err != nil {
		return
	}

	if chipId != CHIPID {
		// No DPS310 detected ... return error
		return fmt.Errorf("dedected wrong chip: %b", chipId)
	}

	err = d.Reset()
	if err != nil {
		return
	}

	d.readCalibration()
	if err != nil {
		return
	}

	// default to high precision
	err = d.ConfigurePressure(RATE_64HZ, PRC_64SAMPLES)
	if err != nil {
		return
	}

	err = d.ConfigureTemperature(RATE_64HZ, PRC_64SAMPLES)
	if err != nil {
		return
	}

	// continuous mode
	err = d.SetMode(ContPresTempMeas)
	if err != nil {
		return
	}

	d.waitDataAvailable(10 * time.Millisecond)

	// set sealevelPressure
	d.seaLevelPressure = 1013.25

	return
}

// waits for pressure and temperature data available
func (d *DPS310) waitDataAvailable(waitTime time.Duration) (err error) {
	var (
		tOk, pOk bool
	)
	for {
		// wait until we have at least one good measurement
		tOk, err = d.TemperatureAvailable()
		if err != nil {
			return
		}
		pOk, err = d.PressureAvailable()
		if err != nil {
			return
		}
		if tOk && pOk {
			break
		}
		time.Sleep(waitTime)
	}
	return nil
}

func (d *DPS310) SetMode(mode Mode) error {

	// Check if mode is possible
	if mode == Invalid1 || mode == Invalid2 {
		return fmt.Errorf("invalid measuring mode: %v", mode)
	}
	if d.pressureOs+d.tempOs > 0 && mode < ContPresMeas {
		return fmt.Errorf("incompatible measuring mode: %v for sample rates > 1", mode)
	}

	data, err := d.readRegister(MEAS_CFG)
	if err != nil {
		return err
	}

	data &= MEAS_CFG_CTRL_CLR_MSK
	mode &=  MEAS_CFG_CTRL_SET_MSK
	data = data | uint8(mode)
	return d.writeRegister(MEAS_CFG, data)
}

// readRegister reads from a single  register.
func (d *DPS310) readRegister(address uint8) (uint8, error) {
	buf := d.rwBuf[:2]
	buf[0] = SPIReadBit| address
	buf[1] = 0
	err := d.tx(buf)
	if err != nil {
		return 0x0, err
	}
	return buf[1], nil
}

// writeRegister writes a single byte to a register.
func (d *DPS310) writeRegister(address, data uint8) (err error) {
	buf := d.rwBuf[:2]
	buf[0] = address
	buf[1] = data
	return d.tx(buf)
}

func (d *DPS310) tx(buf []uint8) (err error) {
	d.cs.Low()
	time.Sleep(CSB_Setup_Time * time.Nanosecond)

	err = d.spi.Tx(buf, buf)

	time.Sleep(CSB_Setup_Time * time.Nanosecond)
	d.cs.High()
	return
}

func (d *DPS310) Reset() (err error) {

	// check func for sesnor ready
	sensorRdy :=
		func() (bool, error) {
			v, err := d.readRegister(MEAS_CFG)
			if err != nil {
				return false, err
			}
			return v & MEAS_CFG_SENSOR_RDY_MSK != 0, nil
		}

	// err = d.writeRegister(RESET, 0b1000_1001)
	err = d.writeRegister(RESET, RESET_VAL)
	if err != nil {
		return
	}

	// Wait for a bit after reset
	time.Sleep(10 * time.Millisecond)

	// Wait for Sensor getting ready
	rdy := false
	for {
		rdy, err = sensorRdy()
		if err != nil {
			return
		}
		if rdy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return
}

func (d *DPS310) readCalibration() (err error) {

	// check func for calibration ready
	calibRdy := func() (bool, error) {
		v, err := d.readRegister(MEAS_CFG)
		if err != nil {
			return false, err
		}
		return v & MEAS_CFG_COEF_RDY_MSK != 0, nil
	}

	// Wait till we're ready to read calibration
	rdy := false
	for {
		rdy, err = calibRdy()
		if err != nil {
			return
		}
		if rdy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	var (
		coeffs [18]uint8
	)
	i := 0
	for addr := uint8(COEF_START); addr < COEF_END; addr++ {
		coeffs[i], err = d.readRegister(addr)
		if err != nil {
			return
		}
		i++
	}

	// see: datasheet page 37
	d.c0 = decVal((uint12_t(coeffs[0])<<4 |
		(uint12_t(coeffs[1]) >> 4 & 0x0f)))

	d.c1 = decVal((uint12_t(coeffs[1])&0xf)<<8 |
		uint12_t(coeffs[2]))

	d.c00 = decVal((uint20_t(coeffs[3]) << 12) |
		(uint20_t(coeffs[4]) << 4) |
		((uint20_t(coeffs[5]) >> 4) & 0x0F))

	d.c10 = decVal(((uint20_t(coeffs[5]) & 0x0f) << 16) |
		(uint20_t(coeffs[6]) << 8) |
		uint20_t(coeffs[7]))

	d.c01 = decVal((uint16_t(coeffs[8]) << 8) |
		uint16_t(coeffs[9]))

	d.c11 = decVal((uint16_t(coeffs[10]) << 8) |
		uint16_t(coeffs[11]))

	d.c20 = decVal((uint16_t(coeffs[12]) << 8) |
		uint16_t(coeffs[13]))

	d.c21 = decVal((uint16_t(coeffs[14]) << 8) |
		uint16_t(coeffs[15]))

	d.c30 = decVal((uint16_t(coeffs[16]) << 8) |
		uint16_t(coeffs[17]))

	return
}

type uint12_t uint32
type uint16_t uint32
type uint20_t uint32
type uint24_t uint32

type coefs interface {
	uint12_t | uint16_t | uint20_t | uint24_t
}

func decVal[T coefs](v T) float32 {
	const (
		max12 = 1<<11 - 1
		sub12 = 1 << 12
		max16 = 1<<15 - 1
		sub16 = 1 << 16
		max20 = 1<<19 - 1
		sub20 = 1 << 20
		max24 = 1<<23 - 1
		sub24 = 1 << 24
	)

	var (
		max, sub int32
	)

	switch any(v).(type) {
	case uint12_t:
		max = max12
		sub = sub12
	case uint16_t:
		max = max16
		sub = sub16
	case uint20_t:
		max = max20
		sub = sub20
	case uint24_t:
		max = max24
		sub = sub24
	}

	val := int32(v)
	if val > max {
		val -= sub
	}
	return float32(val)
}

// Set the sample rate and oversampling averaging for pressure and temperature
// rate How many samples per second to take
// param os How many oversamples to average
func (d *DPS310) ConfigureValues(rate Rate, os Precision) {
	d.ConfigurePressure(rate, os)
	d.ConfigureTemperature(rate, os)
}

func (d *DPS310) isOkRateOs() bool {
	return float32(d.pressureRate)*osScalingData[d.pressureOs].measTime+
		float32(d.tempRate)*osScalingData[d.tempOs].measTime >= 1.0
}

// Set the sample rate and oversampling averaging for pressure
// rate How many samples per second to take
// param os How many oversamples to average
func (d *DPS310) ConfigurePressure(rate Rate, os Precision) (err error) {
	// check rate oversamling combination
	d.pressureRate = rate
	d.pressureOs = os
	if !d.isOkRateOs() {
		return fmt.Errorf("invalid rate oversampling comination")
	}

	data := (uint8(rate) & PRS_CFG_PM_RATE_MSK)<<4 | 
	(uint8(os) & PRS_CFG_PM_PRC_MSK)
	err = d.writeRegister(PRS_CFG, data)
	if err != nil {
		return
	}

	data, err = d.readRegister(CFG_REG)
	if err != nil {
		return
	}

	if os > PRC_8SAMPLES {
		err = d.writeRegister(CFG_REG, data | CFG_REG_PRS_SHIFT_EN_MSK)
	} else {
		err = d.writeRegister(CFG_REG, data & CFG_REG_PRS_SHIFT_DIS_MSK)
	}
	if err != nil {
		return
	}

	d.pressureScale = osScalingData[os].factor

	return
}

// Set the sample rate and oversampling averaging for temperature
// param rate How many samples per second to take
// param os How many oversamples to average
func (d *DPS310) ConfigureTemperature(rate Rate, os Precision) (err error) {
	// find temperatur src
	var (
		data uint8
	)
	data, err = d.readRegister(COEF_SRCE)
	data &= COEF_SRCE_TMP_COEF_SR_MSK

	// check rate oversamling combination
	d.tempRate = rate
	d.tempOs = os
	if !d.isOkRateOs() {
		return fmt.Errorf("invalid rate oversampling comination")
	}

	// add temp rate and oversampling
	data += (uint8(rate)&TMP_CFG_TM_PRC_MSK)<<4 | (uint8(os) & TMP_CFG_TM_RATE_MSK)
	err = d.writeRegister(TMP_CFG, data)
	if err != nil {
		return
	}

	data, err = d.readRegister(CFG_REG)
	if err != nil {
		return
	}

	if os > PRC_8SAMPLES {
		err = d.writeRegister(CFG_REG, data| CFG_REG_TMP_SHIFT_EN_MSK)
	} else {
		err = d.writeRegister(CFG_REG, data& CFG_REG_TMP_SHIFT_DIS_MSK)
	}
	if err != nil {
		return
	}

	d.tempScale = osScalingData[os].factor
	return
}

// Read the temperatur
func (d *DPS310) ReadTemperature() (temperature float32, err error) {

	rawTemperature, err := d.readRawTemperature()
	if err != nil {
		return
	}

	// see: datasheet pages 14 and 15 
	scaledRawtemp := rawTemperature / d.tempScale
	temperature = scaledRawtemp*d.c1 + d.c0/2.0
	return
}

// Read the values  from the sensor
func (d *DPS310) ReadPressure() (pressure, temperature float32, err error) {

	rawPressure, err := d.readRawPressure()
	if err != nil {
		return
	}

	rawTemperature, err := d.readRawTemperature()
	if err != nil {
		return
	}

	// see: datasheet pages 14 and 15 
	scaledRawtemp := rawTemperature / d.tempScale
	temperature = scaledRawtemp*d.c1 + d.c0/2.0

	scaledRawpres := rawPressure / d.pressureScale
	pressure =
		d.c00 + scaledRawpres*(d.c10+
			scaledRawpres*(d.c20+scaledRawpres*d.c30)) +
			scaledRawtemp*d.c01 +
			scaledRawpres*scaledRawtemp*(d.c11+scaledRawpres*d.c21)

	return
}

// reads raw temperature
func (d *DPS310) readRawTemperature() (temperature float32, err error) {
	buf := d.rwBuf[:4]
	buf[0] = 0x80 | TMP_B2
	buf[1] = 0
	buf[2] = 0
	buf[3] = 0
	err = d.tx(buf)
	if err != nil {
		return
	}

	temperature = decVal(uint24_t(buf[1])<<16 |
		uint24_t(buf[2])<<8 |
		uint24_t(buf[3]))
	return
}

// reads raw pressure
func (d *DPS310) readRawPressure() (pressure float32, err error) {
	buf := d.rwBuf[:4]
	buf[0] = SPIReadBit | PRS_B2
	buf[1] = 0
	buf[2] = 0
	buf[3] = 0
	err = d.tx(buf)
	if err != nil {
		return
	}
	pressure = decVal(uint24_t(buf[1])<<16 |
		uint24_t(buf[2])<<8 |
		uint24_t(buf[3]))
	return
}

// Whether new temperature data is available
// returns True if new data available to read
func (d *DPS310) TemperatureAvailable() (bool, error) {
	v, err := d.readRegister(MEAS_CFG)
	return v&MEAS_CFG_TMP_RDY_MSK!= 0, err
}

// Whether new pressure data is available
// returns True if new data available to read
func (d *DPS310) PressureAvailable() (bool, error) {
	v, err := d.readRegister(MEAS_CFG)
	return v&MEAS_CFG_PRS_RDY_MSK != 0, err
}

func (d *DPS310) ReadAltitude() (altitude float32, err error) {
	var (
		p float32
	)

	p0 := d.seaLevelPressure

	p, _, err = d.ReadPressure()
	if err != nil {
		return
	}

	altitude = float32(44330.0 *
		(1.0 - math.Pow(float64(((p/100.)/p0)), 0.1903)))
	return
}

// calibrate sealevelPressure to given height
func (d *DPS310) CalibrateAltitude(h float32) (err error) {
	var (
		p float32
	)
	p, _, err = d.ReadPressure()
	if err != nil {
		return
	}
	d.seaLevelPressure = p / 100.0 * float32(math.Pow((1.-float64(h)/44330.0), -5.255))
	return
}
