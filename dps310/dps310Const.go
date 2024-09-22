package dps310

const (
	SPIReadBit = 0b1000_0000
)

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
	RATE_MAXRATE
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
	CHIPID          			= 0x10
	CSB_Setup_Time   			= 20 // (Nanoseconds)

	// register and masks
	PRS_B2           			= 0x00
	TMP_B2           			= 0x03
	PRS_CFG         	 		= 0x06
	PRS_CFG_PM_RATE_MSK 		= 0b0000_0111
	PRS_CFG_PM_PRC_MSK 			= 0b0000_1111
	TMP_CFG        				= 0x07
	TMP_CFG_TM_RATE_MSK 		= 0b0000_0111
	TMP_CFG_TM_PRC_MSK 			= 0b0000_1111
	MEAS_CFG       				= 0x08
	MEAS_CFG_COEF_RDY_MSK 		= 0b1000_0000
	MEAS_CFG_SENSOR_RDY_MSK 	= 0b0100_0000
	MEAS_CFG_TMP_RDY_MSK 		= 0b0010_0000
	MEAS_CFG_PRS_RDY_MSK 		= 0b0001_0000
	MEAS_CFG_CTRL_CLR_MSK   	= 0b1111_1000
	MEAS_CFG_CTRL_SET_MSK   	= 0b0000_0111
	CFG_REG      				= 0x09
	CFG_REG_TMP_SHIFT_EN_MSK	= 0b0000_1000
	CFG_REG_TMP_SHIFT_DIS_MSK	= 0b1111_0111
	CFG_REG_PRS_SHIFT_EN_MSK	= 0b0000_0100
	CFG_REG_PRS_SHIFT_DIS_MSK	= 0b1111_1011
	COEF_START  		 		= 0x10
	COEF_END     				= 0x22
	RESET       				= 0x0C
	RESET_VAL    				= 0b1000_1001
	Product_ID   				= 0x0D
	COEF_SRCE 					= 0x28
	COEF_SRCE_TMP_COEF_SR_MSK 	= 0b1000_0000
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
