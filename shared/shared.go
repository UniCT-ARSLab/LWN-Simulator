package shared

import "log"

// Verbose flag
var Verbose bool = false

func DebugPrint(msg string) {
	if Verbose {
		log.Printf("[DEBUG]: %s", msg)
	}
}
