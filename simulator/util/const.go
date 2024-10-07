package util

// Constants
const (
	ConnectedPowerSource = 0

	MAXFCNTGAP = uint32(16384)

	PrintBoth = iota
	PrintOnlySocket
	PrintOnlyConsole

	Stopped
	Running

	Normal
	Retransmission
	FPending
	Activation
)
