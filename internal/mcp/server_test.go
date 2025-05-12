package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServerInit(t *testing.T) {
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server.PidFile == "" {
		t.Error("PidFile path is empty")
	}

	if server.LogFile == "" {
		t.Error("LogFile path is empty")
	}
}

func TestServerMethods(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mcp-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test server with test files
	server := &Server{
		PidFile: filepath.Join(tmpDir, "mcp.pid"),
		LogFile: filepath.Join(tmpDir, "mcp.log"),
	}

	// Test IsRunning when no PID file exists
	if server.IsRunning() {
		t.Error("Server should not be running when no PID file exists")
	}

	// Test getPid when no PID file exists
	_, err = server.getPid()
	if err == nil {
		t.Error("getPid should return error when PID file does not exist")
	}

	// Write an invalid PID file
	err = os.WriteFile(server.PidFile, []byte("not-a-number"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test PID file: %v", err)
	}

	// Test getPid with invalid PID
	_, err = server.getPid()
	if err == nil {
		t.Error("getPid should return error with invalid PID")
	}

	// Test status when server is not running
	status := server.Status()
	if status != "MCP server is not running" {
		t.Errorf("Unexpected status: %s", status)
	}

	// Stop when server is not running
	err = server.Stop()
	if err == nil {
		t.Error("Stop should return error when server is not running")
	}
}

// Only run this test manually as it involves starting an actual process
func TestServerLifecycle(t *testing.T) {
	if os.Getenv("RUN_MANUAL_TESTS") != "1" {
		t.Skip("Skipping manual test. Set RUN_MANUAL_TESTS=1 to run.")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mcp-lifecycle")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test server with test files
	server := &Server{
		PidFile: filepath.Join(tmpDir, "mcp.pid"),
		LogFile: filepath.Join(tmpDir, "mcp.log"),
	}

	// Start the server
	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Check that the server is running
	if !server.IsRunning() {
		t.Error("Server should be running after start")
	}

	// Check that the PID file exists
	if _, err := os.Stat(server.PidFile); os.IsNotExist(err) {
		t.Error("PID file should exist after start")
	}

	// Restart the server
	err = server.Restart()
	if err != nil {
		t.Fatalf("Failed to restart server: %v", err)
	}

	// Check that the server is still running
	if !server.IsRunning() {
		t.Error("Server should be running after restart")
	}

	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Give it a moment to fully shut down
	time.Sleep(1 * time.Second)

	// Check that the server is no longer running
	if server.IsRunning() {
		t.Error("Server should not be running after stop")
	}

	// Check that the PID file was removed
	if _, err := os.Stat(server.PidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after stop")
	}
}
