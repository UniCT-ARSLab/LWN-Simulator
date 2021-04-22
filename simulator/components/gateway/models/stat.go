package models

type Stat struct {
	RXNb uint32  `json:"-"` // Number of radio packets received (unsigned integer)
	RXOK uint32  `json:"-"` // Number of radio packets received with a valid PHY CRC
	RXFW uint32  `json:"-"` // Number of radio packets forwarded (unsigned integer)
	ACKR float64 `json:"-"` // Percentage of upstream datagrams that were acknowledged
	DWNb uint32  `json:"-"` // Number of downlink datagrams received (unsigned integer)
	TXNb uint32  `json:"-"` // Number of packets emitted (unsigned integer)
}
