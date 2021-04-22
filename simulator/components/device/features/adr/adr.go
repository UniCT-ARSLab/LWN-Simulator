package adr

const (
	//ADRACKLIMIT is max value to ConfUplink whitout ack
	ADRACKLIMIT = int8(64)
	//ADRACKDELAY is max delay to ConfUplink whitout ack after ADRACKLIMIT
	ADRACKDELAY = int8(32)
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
func (adr *ADRInfo) Reset() {
	adr.ADRACKCnt = 0
	adr.ADRACKReq = false
}
