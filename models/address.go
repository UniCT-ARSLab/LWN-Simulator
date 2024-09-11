package models

// AddressIP represents an IP address and associated port.
type AddressIP struct {
	Address string `json:"ip"`   // IP address in dotted-quad notation
	Port    string `json:"port"` // Port number as a string
}
