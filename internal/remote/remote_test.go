package remote

import (
	"testing"
)

func TestValidateGitURL(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid HTTPS URLs
		{"GitHub HTTPS with .git", "https://github.com/user/repo.git", false},
		{"GitHub HTTPS without .git", "https://github.com/user/repo", false},
		{"GitLab HTTPS with .git", "https://gitlab.com/user/repo.git", false},
		{"GitLab HTTPS without .git", "https://gitlab.com/user/repo", false},
		{"Bitbucket HTTPS", "https://bitbucket.org/user/repo.git", false},
		{"Codeberg HTTPS", "https://codeberg.org/user/repo.git", false},
		{"SourceHut HTTPS", "https://git.sr.ht/~user/repo", false},

		// Valid SSH URLs
		{"GitHub SSH", "git@github.com:user/repo.git", false},
		{"GitLab SSH", "git@gitlab.com:user/repo.git", false},
		{"Custom host SSH", "git@git.example.com:user/repo.git", false},

		// Valid unknown host with .git
		{"Unknown host with .git", "https://git.example.com/user/repo.git", false},

		// Invalid URLs
		{"Empty URL", "", true},
		{"Invalid scheme", "ftp://github.com/user/repo.git", true},
		{"No protocol", "github.com/user/repo.git", true},
		{"Known host without proper path", "https://github.com/invalid", true},
		{"Known host with too many path segments", "https://github.com/user/repo/extra/path", true},
		{"Unknown host without .git", "https://git.example.com/user/repo", true},
		{"Invalid SSH format", "git@github.com/user/repo.git", true},
		{"Malformed URL", "https://[invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateGitURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGitURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
