package display

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintProjectHeader(t *testing.T) {
	output := captureOutput(PrintProjectHeader)
	if !strings.Contains(output, "PROJECTS:") {
		t.Errorf("Expected output to contain 'PROJECTS:', got %s", output)
	}
}

func TestPrintCommandHeader(t *testing.T) {
	output := captureOutput(PrintCommandHeader)
	if !strings.Contains(output, "COMMANDS:") {
		t.Errorf("Expected output to contain 'COMMANDS:', got %s", output)
	}
}

func TestPrintProjectName(t *testing.T) {
	output := captureOutput(func() {
		PrintProjectName("test-project")
	})
	if !strings.Contains(output, "📁 Name: test-project") {
		t.Errorf("Expected output to contain '📁 Name: test-project', got %s", output)
	}
}

func TestPrintCommandName(t *testing.T) {
	output := captureOutput(func() {
		PrintCommandName("test-command")
	})
	if !strings.Contains(output, "⚡ Name: test-command") {
		t.Errorf("Expected output to contain '⚡ Name: test-command', got %s", output)
	}
}

func TestPrintNoItemsFound(t *testing.T) {
	output := captureOutput(func() {
		PrintNoItemsFound("projects")
	})
	if !strings.Contains(output, "No projects found.") {
		t.Errorf("Expected output to contain 'No projects found.', got %s", output)
	}
}
