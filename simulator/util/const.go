package util

const (
	ConnectedPowerSource = 0

	MAXFCNTGAP = uint32(16384)

	PrintBoth = iota
	PrintOnlySocket
	PrintOnlyConsole

	//stati del simulatore
	Stopped = iota
	Running
)
