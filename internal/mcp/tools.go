package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// ToolsClient represents a client for the MCP server's tools
type ToolsClient struct {
	BaseURL string
	Client  *http.Client
}

// SSEHandler defines a function that handles SSE events
type SSEHandler func(event string, data string)

// NewToolsClient creates a new client for the MCP server
func NewToolsClient() *ToolsClient {
	return &ToolsClient{
		BaseURL: "http://localhost:8080",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetHealth checks if the MCP server is running
func (c *ToolsClient) GetHealth() (ToolResponse, error) {
	var response ToolResponse

	// Make request to health endpoint
	resp, err := c.Client.Get(c.BaseURL + "/health")
	if err != nil {
		return response, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// ListCommands gets all available commands
func (c *ToolsClient) ListCommands() (ToolResponse, error) {
	var response ToolResponse

	// Make request to commands endpoint
	resp, err := c.Client.Get(c.BaseURL + "/commands")
	if err != nil {
		return response, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// ExecuteCommand runs a command on the MCP server
func (c *ToolsClient) ExecuteCommand(name string, args map[string]interface{}) (CommandResponse, error) {
	var response CommandResponse

	// Prepare request body
	reqBody := CommandRequest{
		Name: name,
		Args: args,
	}

	// Convert to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return response, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to execute endpoint
	resp, err := c.Client.Post(c.BaseURL+"/commands/execute", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return response, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// ListTools gets all available tools
func (c *ToolsClient) ListTools() (ToolResponse, error) {
	var response ToolResponse

	// Make request to tools endpoint
	resp, err := c.Client.Get(c.BaseURL + "/tools/list")
	if err != nil {
		return response, fmt.Errorf("failed to connect to MCP server: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}

// SubscribeToEvents connects to the SSE endpoint and calls the handler for each event
func (c *ToolsClient) SubscribeToEvents(handler SSEHandler) error {
	maxRetries := 5
	retryCount := 0
	var lastRetry time.Time

	// Define possible SSE endpoints to try in order
	sseEndpoints := []string{"/events", "/sse", "/mcp"}

	for {
		// Check if we've exceeded max retries
		if retryCount >= maxRetries {
			return fmt.Errorf("exceeded maximum retry attempts (%d)", maxRetries)
		}

		// Add exponential backoff for retries
		if retryCount > 0 {
			backoffTime := time.Duration(math.Pow(2, float64(retryCount-1))) * time.Second
			if time.Since(lastRetry) < backoffTime {
				sleepTime := backoffTime - time.Since(lastRetry)
				fmt.Printf("Waiting %v before retry attempt %d/%d...\n", sleepTime, retryCount+1, maxRetries)
				time.Sleep(sleepTime)
			}
			lastRetry = time.Now()
		}

		// Try each possible endpoint until one works
		var lastErr error
		var connected bool

		for _, endpoint := range sseEndpoints {
			// Log connection attempt
			fmt.Printf("Connecting to SSE endpoint: %s%s (attempt %d/%d)\n", c.BaseURL, endpoint, retryCount+1, maxRetries)

			// Create a longer-lived context for the initial connection
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

			// Create a transport that properly supports streaming with longer timeouts
			transport := &http.Transport{
				ResponseHeaderTimeout: 60 * time.Second,
				IdleConnTimeout:       120 * time.Second,
				DisableCompression:    true,
			}

			// Create a client without a timeout for the SSE connection itself
			client := &http.Client{
				Transport: transport,
				Timeout:   0, // No timeout for the SSE connection
			}

			// Make request to events endpoint
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+endpoint, nil)
			if err != nil {
				cancel()
				lastErr = fmt.Errorf("failed to create request: %w", err)
				continue // Try next endpoint
			}

			// Set headers for SSE
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Connection", "keep-alive")

			// Send request
			resp, err := client.Do(req)
			if err != nil {
				cancel()
				fmt.Printf("Connection error for %s: %v\n", endpoint, err)
				lastErr = err
				continue // Try next endpoint
			}

			// Check response status
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				cancel()
				fmt.Printf("Server returned error for %s: HTTP %d - %s\n", endpoint, resp.StatusCode, string(body))
				lastErr = fmt.Errorf("SSE error: Non-200 status code (%d)", resp.StatusCode)
				continue // Try next endpoint
			}

			// Verify content type
			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "text/event-stream") {
				resp.Body.Close()
				cancel()
				fmt.Printf("Invalid content type for %s: %s (expected text/event-stream)\n", endpoint, contentType)
				lastErr = fmt.Errorf("invalid content type: %s", contentType)
				continue // Try next endpoint
			}

			// Connection successful
			connected = true

			// Reset retry count since we've established a connection
			retryCount = 0
			fmt.Printf("Successfully connected to event stream at %s. Waiting for events...\n", endpoint)

			// Create a separate context for the event reading loop
			readCtx, readCancel := context.WithCancel(context.Background())

			// Start a goroutine to cancel the read context when the request context is done
			go func() {
				<-ctx.Done()
				readCancel()
			}()

			// Process SSE stream
			scanner := bufio.NewScanner(resp.Body)
			var eventName string
			var dataBuffer strings.Builder

			// Use a done channel to signal when the scanner exits
			done := make(chan struct{})

			go func() {
				defer close(done)

				for scanner.Scan() {
					select {
					case <-readCtx.Done():
						return
					default:
						line := scanner.Text()

						// Empty line marks the end of an event
						if line == "" {
							if eventName != "" && dataBuffer.Len() > 0 {
								// Process the complete event
								handler(eventName, dataBuffer.String())

								// Reset for next event
								eventName = ""
								dataBuffer.Reset()
							}
							continue
						}

						// Parse the line
						if strings.HasPrefix(line, "event:") {
							eventName = strings.TrimSpace(line[6:])
						} else if strings.HasPrefix(line, "data:") {
							// Append to data buffer (could be multiple data lines per event)
							if dataBuffer.Len() > 0 {
								dataBuffer.WriteString("\n")
							}
							dataBuffer.WriteString(strings.TrimSpace(line[5:]))
						}
					}
				}
			}()

			// Wait for scanner to complete or context to be canceled
			select {
			case <-readCtx.Done():
				resp.Body.Close()
				cancel()
				fmt.Println("Connection closed by client")
				return nil
			case <-done:
				// Scanner exited
				resp.Body.Close()
				readCancel()

				// Check for scanner error
				if err := scanner.Err(); err != nil {
					fmt.Printf("Stream error: %v\n", err)
					// Don't treat EOF as an error that requires retry
					if !strings.Contains(err.Error(), "EOF") {
						retryCount++
					}
				} else {
					fmt.Println("Stream closed by server")
				}

				cancel()
			}

			// Break out of the endpoint loop if we successfully connected
			break
		}

		// If we didn't connect to any endpoint, increment retry counter
		if !connected {
			retryCount++
			if lastErr != nil {
				fmt.Printf("All SSE endpoints failed. Last error: %v. Retrying...\n", lastErr)
			}
		}

		// Wait a second before attempting to reconnect
		time.Sleep(time.Second)
	}
}
