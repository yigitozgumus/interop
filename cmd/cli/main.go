package main

import (
	"fmt"
	"interop/internal/settings"
	"interop/internal/util"
	"log"
)

// These variables are set during build time using ldflags
var (
	version    = "dev"
	isSnapshot = "false"
)

func main() {
	_, err := settings.Load()
	if err != nil {
		log.Fatalf("settings init: %v", err)
	}
	util.Message("Config is loaded")

	// 2. Wire up anything that depends on settings.
	//initLogger(cfg.LogLevel)

	// 4. Hand off to the rest of your CLI.
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
