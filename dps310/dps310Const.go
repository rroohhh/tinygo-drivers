package dps310

type Rate uint8

const (
	RATE_1HZ Rate = iota
	RATE_2HZ
	RATE_4HZ
	RATE_8HZ
	RATE_16HZ
	RATE_32HZ
	RATE_64HZ
	RATE_128HZ
)

type Precision uint8

const (
	PRC_1SAMPLE Precision = iota
	PRC_2SAMPLES
	PRC_4SAMPLES
	PRC_8SAMPLES
	PRC_16SAMPLES
	PRC_32SAMPLES
	PRC_64SAMPLES
	PRC_128SAMPLES
	PRC_MAXSAMPLES
)

type Mode uint8

const (
	Idle             Mode = 0b0000_00000
	PresMeas         Mode = 0b0000_00001
	TempMeas         Mode = 0b0000_00010
	Invalid1         Mode = 0b0000_00011
	Invalid2         Mode = 0b0000_00100
	ContPresMeas     Mode = 0b0000_00101
	ContTempMeas     Mode = 0b0000_00110
	ContPresTempMeas Mode = 0b0000_00111
)

const (
	CHIPID          = 0x10
	I2CADDR_DEFAULT = 119
	// register adresses
	PRSB2       = 0x00
	TMPB2       = 0x03
	PRSCFG      = 0x06
	TMPCFG      = 0x07
	MEASCFG     = 0x08
	CFGREG      = 0x09
	COEFSTART   = 0x10
	COEFEND     = 0x22
	RESET       = 0x0C
	PRODREVID   = 0x0D
	TMPCOEFSRCE = 0x28
	SetUpTime   = 20 // CSB Setup Time (Nanoseconds)
)

type osData struct {
	factor    float32 // scaling factor
	precision float32 // Pa
	measTime  float32 // milliSeconds
}

var (
	osScalingData []osData = []osData{
		{
			factor:    524288,
			precision: 2.5,
			measTime:  3.6,
		},
		{
			factor:    1572864,
			precision: 1,
			measTime:  5.2,
		},
		{
			factor:    3670016,
			precision: 0.5,
			measTime:  8.4,
		},
		{
			factor:    7864320,
			precision: 0.4,
			measTime:  14.8,
		},
		{
			factor:    253952,
			precision: 0.35,
			measTime:  27.6,
		},
		{
			factor:    516096,
			precision: 0.3,
			measTime:  53.2,
		},
		{
			factor:    1040384,
			precision: 0.2,
			measTime:  104.4,
		},
		{
			factor:    2088960,
			precision: 0.2,
			measTime:  206.8,
		},
	}
)
