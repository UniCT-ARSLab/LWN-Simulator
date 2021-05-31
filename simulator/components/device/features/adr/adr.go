package adr

import (
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
)

const (
	ADRACKLIMIT = int8(64)
	ADRACKDELAY = int8(32)

	CodeNoneError = iota
	CodeADRFlagReqSet
	CodeUnjoined
)

//ADRInfo contains adr bits
type ADRInfo struct {
	ADR       bool `json:"-"`
	ADRACKCnt int8 `json:"-"`
	ADRACKReq bool `json:"-"`
}

//Setup struct
func (adr *ADRInfo) Setup(ADRValue bool) {
	adr.ADR = ADRValue
	adr.ADRACKCnt = 0
	adr.ADRACKReq = false
}

//Reset struct
func (adr *ADRInfo) Reset() string {

	adr.ADRACKCnt = 0

	result := ""
	if adr.ADRACKReq {
		result = "UNSET ADRACKReq flag"
	}
	adr.ADRACKReq = false

	return result
}

func (adr *ADRInfo) ADRProcedure(datarate uint8, region rp.Region, supportedADR bool) (uint8, int) {

	switch adr.ADRACKCnt {

	case ADRACKLIMIT, ADRACKLIMIT + ADRACKDELAY:

		if datarate > region.GetMinDataRate() && supportedADR {
			adr.ADRACKReq = true

			return 0, CodeADRFlagReqSet
		}

	}

	if adr.ADRACKCnt%ADRACKDELAY == 0 && adr.ADRACKCnt > ADRACKLIMIT {

		if datarate > region.GetMinDataRate() {

			datarateNEW := rp.DecrementDataRate(region, datarate)
			return datarateNEW, CodeNoneError

		} else {

			return datarate, CodeUnjoined
		}

	}

	return datarate, CodeNoneError

}
