package edit

import (
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"os"
	"os/exec"
	"path/filepath"
)

// OpenSettings opens the settings file using the specified editor or defaults to $EDITOR environment variable
// If editorName is empty, it will use the editor from $EDITOR environment variable or fall back to nano
func OpenSettings(editorName string) error {
	// Find the settings file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorAndExit("failed to get user home directory: %w", err)
	}

	// Build the path to the settings file using the current path configuration
	config := filepath.Join(homeDir, settings.DefaultPathConfig.SettingsDir)
	base := filepath.Join(config, settings.DefaultPathConfig.AppDir)
	settingsPath := filepath.Join(base, settings.DefaultPathConfig.CfgFile)

	// Determine which editor to use
	var editor string
	if editorName != "" {
		// Use the editor specified via the --editor flag
		editor = editorName
	} else {
		// Fall back to the original behavior: check $EDITOR environment variable
		editor = os.Getenv("EDITOR")
		if editor == "" {
			// Default to common editors if $EDITOR is not set
			editor = "nano" // Simple default that's often available
		}
	}

	logging.Message(fmt.Sprintf("Opening settings file with %s: %s", editor, settingsPath))

	// Create the command to open the editor
	cmd := exec.Command(editor, settingsPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the editor
	return cmd.Run()
}
