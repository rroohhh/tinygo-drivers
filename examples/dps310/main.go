package main

import (
	"fmt"
	"machine"
	"time"

	"tinygo.org/x/drivers/dps310"
)

const (
	DefaultInterval = 1   // second
	Height 			= 329 // meters, adjust for your place
	NumMeas 		= 50
)

func blink(led machine.Pin, t, c int) {
	for i := 0; i < c; i++ {
		led.High()
		time.Sleep(time.Millisecond * time.Duration(t))
		led.Low()
		time.Sleep(time.Millisecond * time.Duration(t))
	}
}

func main() {
	var (
		err error
	)

	// Do some communiction by blinking the led
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})
	blink(led, 750, 3)

	endFunc := func() {
		blink(led, 750, 5)
		fmt.Println("left program")
	}

	interval := time.Duration(DefaultInterval) * time.Second
	fmt.Println("dps310 measuring interval:", interval)
	defer endFunc()

	s, err := dps310.NewSPI(
		dps310.Config{
			SPI: machine.SPI0,
			CSD: machine.GPIO17,
			SPI_Config: machine.SPIConfig{
				Frequency: 10000000,
				LSBFirst:  false,
				Mode:      machine.Mode3,
				SCK:       machine.SPI0_SCK_PIN,
				SDO:       machine.SPI0_SDO_PIN,
				SDI:       machine.SPI0_SDI_PIN,
			},
		})

	if err != nil {
		fmt.Println("configuration Error:", err)
		return
	}

	fmt.Println("Initalizing dps310")
	err = s.Init()
	if err != nil {
		fmt.Println("Init error:", err)
		return
	}

	// do a couple of measurements
	fmt.Println("Measuring pressure and temperature")
	for i := 0; i < NumMeas ; i++ {
		led.High()
		pres, temp, _ := s.ReadPressure()
		fmt.Printf("%v;%v\n", pres, temp)
		led.Low()
		time.Sleep(interval)
	}

	// Reset for new configuration
	err = s.Reset()
	if err != nil {
		fmt.Println("Reset sensor error:", err)
		return
	}

	err = s.ConfigurePressure(dps310.RATE_32HZ, dps310.PRC_4SAMPLES)
	if err != nil {
		fmt.Println("Configure pressure error:", err)
		return
	}

	err = s.ConfigureTemperature(dps310.RATE_32HZ, dps310.PRC_4SAMPLES)
	if err != nil {
		fmt.Println("Configure temperature error:", err)
		return
	}

	err = s.SetMode(dps310.ContPresTempMeas)
	if err != nil {
		fmt.Println("Set mode error:", err)
		return
	}

	// wait until we have at least one good measurement	
	for {
		tOk, err := s.TemperatureAvailable()
		if err != nil {
			fmt.Println("Check temperature available error:", err)
			return
		}
		pOk, err := s.PressureAvailable()
		if err != nil {
			fmt.Println("Check pressure available error:", err)
			return
		}
		if tOk && pOk {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = s.CalibrateAltitude(Height)
	if err != nil {
		fmt.Println("Calibrate altitude error:", err)
		return
	}
	
	fmt.Println("Measuring altitude")
	for i := 0; i < NumMeas ; i++ {
		led.High()
		h, _ := s.ReadAltitude()
		fmt.Printf("%v\n",  h)
		led.Low()
		time.Sleep(interval)
	}
}
