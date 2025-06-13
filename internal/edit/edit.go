package edit

import (
	"fmt"
	"interop/internal/logging"
	"interop/internal/settings"
	"os"
	"os/exec"
	"path/filepath"
)

// OpenConfigFolder opens the entire interop config folder using the best available editor or file browser
func OpenConfigFolder(editorName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.ErrorAndExit("failed to get user home directory: %w", err)
	}

	// Path to the interop config folder
	configDir := filepath.Join(homeDir, settings.DefaultPathConfig.SettingsDir, settings.DefaultPathConfig.AppDir)

	// Determine which editor or opener to use
	var cmd *exec.Cmd

	// 1. If user specified an editor, try to use it
	if editorName != "" {
		if editorName == "code" {
			cmd = exec.Command("code", configDir)
		} else {
			cmd = exec.Command(editorName, configDir)
		}
	} else if _, err := exec.LookPath("code"); err == nil {
		// 2. Prefer VS Code if available
		cmd = exec.Command("code", configDir)
	} else if _, err := exec.LookPath("open"); err == nil {
		// 3. macOS Finder
		cmd = exec.Command("open", configDir)
	} else if _, err := exec.LookPath("xdg-open"); err == nil {
		// 4. Linux file browser
		cmd = exec.Command("xdg-open", configDir)
	} else {
		// 5. Fallback: try $EDITOR or nano
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}
		cmd = exec.Command(editor, configDir)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logging.Message(fmt.Sprintf("Opening config folder: %s", configDir))
	return cmd.Run()
}
