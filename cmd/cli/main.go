package main

import (
	"fmt"
)

// These variables are set during build time using ldflags
var (
	version    = "dev"
	isSnapshot = "false"
)

func main() {
	displayHelp()
}

func displayHelp() {
	versionInfo := version
	if isSnapshot == "true" {
		versionInfo += " (snapshot)"
	}

	fmt.Printf("Interop - Version %s\n\n", versionInfo)
	fmt.Println("Documentation and Help:")
	fmt.Println("  Visit https://github.com/yigitozgumus/interop for more details.")
}
