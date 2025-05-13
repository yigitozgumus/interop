package mcp

import (
	"interop/internal/settings"
	"net/http"
	"sync"
)

// MCPServer is the HTTP server for MCP
type MCPServer struct {
	Port      int
	DataDir   string
	Commands  map[string]settings.CommandConfig
	handlers  map[string]http.HandlerFunc
	clients   map[chan string]bool
	clientsMu sync.Mutex
}

// CommandRequest represents a request to execute a command
type CommandRequest struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// CommandResponse represents the result of a command execution
type CommandResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// ToolResponse represents a response from a tool
type ToolResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SSEEvent represents an event for SSE
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}
