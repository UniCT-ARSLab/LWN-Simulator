package shared

import "log"

// Verbose flag
var Verbose bool = false

// Version of the simulator
const Version = "1.0.3"

func DebugPrint(msg string) {
	if Verbose {
		log.Printf("[DEBUG]: %s", msg)
	}
}
