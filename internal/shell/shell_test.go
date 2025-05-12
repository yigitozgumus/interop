package shell

import (
	"strings"
	"testing"
)

func TestGetShellTypeFromPath(t *testing.T) {
	tests := []struct {
		name      string
		shellPath string
		want      ShellType
	}{
		{
			name:      "Bash shell",
			shellPath: "/bin/bash",
			want:      ShellTypeBash,
		},
		{
			name:      "Zsh shell",
			shellPath: "/bin/zsh",
			want:      ShellTypeZsh,
		},
		{
			name:      "Fish shell",
			shellPath: "/usr/bin/fish",
			want:      ShellTypeFish,
		},
		{
			name:      "Sh shell",
			shellPath: "/bin/sh",
			want:      ShellTypeSh,
		},
		{
			name:      "Unknown shell",
			shellPath: "/bin/unknown",
			want:      ShellTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getShellTypeFromPath(tt.shellPath); got != tt.want {
				t.Errorf("getShellTypeFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAliasCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{
			name: "Alias command",
			cmd:  "alias:my-alias",
			want: true,
		},
		{
			name: "Not an alias command",
			cmd:  "echo hello",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAliasCommand(tt.cmd); got != tt.want {
				t.Errorf("IsAliasCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLocalScriptCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{
			name: "Local script command",
			cmd:  "./myscript.sh",
			want: true,
		},
		{
			name: "Local script with args",
			cmd:  "./myscript.sh arg1 arg2",
			want: true,
		},
		{
			name: "Not a local script",
			cmd:  "echo hello",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLocalScriptCommand(tt.cmd); got != tt.want {
				t.Errorf("IsLocalScriptCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseLocalScript(t *testing.T) {
	tests := []struct {
		name       string
		cmd        string
		wantScript string
		wantArgs   []string
	}{
		{
			name:       "Script without args",
			cmd:        "./myscript.sh",
			wantScript: "./myscript.sh",
			wantArgs:   nil,
		},
		{
			name:       "Script with args",
			cmd:        "./myscript.sh arg1 arg2",
			wantScript: "./myscript.sh",
			wantArgs:   []string{"arg1", "arg2"},
		},
		{
			name:       "Empty command",
			cmd:        "",
			wantScript: "",
			wantArgs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotScript, gotArgs := ParseLocalScript(tt.cmd)
			if gotScript != tt.wantScript {
				t.Errorf("ParseLocalScript() script = %v, want %v", gotScript, tt.wantScript)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("ParseLocalScript() args length = %v, want %v", len(gotArgs), len(tt.wantArgs))
				return
			}

			for i, arg := range gotArgs {
				if arg != tt.wantArgs[i] {
					t.Errorf("ParseLocalScript() arg[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestShellExecuteCommand(t *testing.T) {
	shell := Shell{
		Path: "/bin/sh",
		Type: ShellTypeSh,
	}

	cmd := shell.ExecuteCommand("echo hello")

	if cmd.Path != "/bin/sh" {
		t.Errorf("ExecuteCommand() path = %v, want %v", cmd.Path, "/bin/sh")
	}

	if !strings.Contains(strings.Join(cmd.Args, " "), "echo hello") {
		t.Errorf("ExecuteCommand() args = %v, should contain 'echo hello'", cmd.Args)
	}
}

func TestShellExecuteAlias(t *testing.T) {
	shell := Shell{
		Path: "/bin/bash",
		Type: ShellTypeBash,
	}

	cmd := shell.ExecuteAlias("alias:my-alias")

	if cmd.Path != "/bin/bash" {
		t.Errorf("ExecuteAlias() path = %v, want %v", cmd.Path, "/bin/bash")
	}

	if !strings.Contains(strings.Join(cmd.Args, " "), "my-alias") {
		t.Errorf("ExecuteAlias() args = %v, should contain 'my-alias'", cmd.Args)
	}
}
