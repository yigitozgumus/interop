package mcp

import (
	"encoding/json"
	"testing"
)

func TestFormatToolOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ToolOutput
	}{
		{
			name:  "simple text",
			input: "Hello, World!",
			expected: ToolOutput{
				Type: "text",
				Text: "Hello, World!",
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: ToolOutput{
				Type: "text",
				Text: "",
			},
		},
		{
			name:  "multiline text",
			input: "Line 1\nLine 2\nLine 3",
			expected: ToolOutput{
				Type: "text",
				Text: "Line 1\nLine 2\nLine 3",
			},
		},
		{
			name:  "text with special characters",
			input: `Text with "quotes" and \backslashes`,
			expected: ToolOutput{
				Type: "text",
				Text: `Text with "quotes" and \backslashes`,
			},
		},
		{
			name:  "JSON content",
			input: `{"key": "value", "number": 42}`,
			expected: ToolOutput{
				Type: "text",
				Text: `{"key": "value", "number": 42}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatToolOutput(tt.input)

			// Parse the result to verify it's valid JSON
			var parsed ToolOutput
			err := json.Unmarshal([]byte(result), &parsed)
			if err != nil {
				t.Errorf("formatToolOutput() returned invalid JSON: %v", err)
				return
			}

			// Verify the structure
			if parsed.Type != tt.expected.Type {
				t.Errorf("formatToolOutput() Type = %v, want %v", parsed.Type, tt.expected.Type)
			}

			if parsed.Text != tt.expected.Text {
				t.Errorf("formatToolOutput() Text = %v, want %v", parsed.Text, tt.expected.Text)
			}
		})
	}
}

func TestFormatToolOutput_ValidJSON(t *testing.T) {
	// Test that the output is always valid JSON
	inputs := []string{
		"Simple text",
		"",
		"Text with\nnewlines",
		`Text with "quotes"`,
		"Text with\ttabs",
		"Text with unicode: 🚀 ñ é",
		"Very long text " + string(make([]byte, 1000)), // Large text
	}

	for _, input := range inputs {
		result := formatToolOutput(input)

		// Verify it's valid JSON
		var parsed interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Errorf("formatToolOutput(%q) produced invalid JSON: %v", input, err)
		}
	}
}
