package edit

import (
	"fmt"
	"interop/internal/settings"
	"interop/internal/util"
	"os"
	"os/exec"
	"path/filepath"
)

// OpenSettings opens the settings file using the editor specified in $EDITOR environment variable
func OpenSettings() error {
	// Find the settings file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		util.Error("failed to get user home directory: %w", err)
	}

	// Build the path to the settings file using the current path configuration
	config := filepath.Join(homeDir, settings.DefaultPathConfig.SettingsDir)
	base := filepath.Join(config, settings.DefaultPathConfig.AppDir)
	settingsPath := filepath.Join(base, settings.DefaultPathConfig.CfgFile)

	// Get the editor from environment variable
	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Default to common editors if $EDITOR is not set
		editor = "nano" // Simple default that's often available
	}

	util.Message(fmt.Sprintf("Opening settings file with %s: %s", editor, settingsPath))

	// Create the command to open the editor
	cmd := exec.Command(editor, settingsPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor
	return cmd.Run()
}
