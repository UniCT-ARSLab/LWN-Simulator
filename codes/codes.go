package codes

// A set of status codes
const (
    // CodeOK indicates the operation was successful.
    CodeOK = iota
    // CodeErrorName indicates there was an error with the name.
    CodeErrorName
    // CodeErrorAddress indicates there was an error with the address.
    CodeErrorAddress
    // CodeErrorDeviceActive indicates the device is currently active and cannot be modified.
    CodeErrorDeviceActive
    // CodeNoBridge indicates that no bridge was found.
    CodeNoBridge
    // CodeErrorGatewayActive indicates that the gateway is currently active and cannot be modified.
    CodeErrorGatewayActive
    // CodeSaving indicates that the operation is saving data.
    CodeSaving
)
