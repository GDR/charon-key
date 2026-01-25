package config

import (
	"testing"
)

func TestParseUserMap(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      map[string][]string
		wantError bool
	}{
		{
			name:  "single mapping",
			input: "alice:alice-github",
			want: map[string][]string{
				"alice": {"alice-github"},
			},
			wantError: false,
		},
		{
			name:  "multiple mappings same SSH user",
			input: "alice:alice-github,alice:shared-github",
			want: map[string][]string{
				"alice": {"alice-github", "shared-github"},
			},
			wantError: false,
		},
		{
			name:  "multiple SSH users",
			input: "alice:alice-github,bob:bob-github",
			want: map[string][]string{
				"alice": {"alice-github"},
				"bob":   {"bob-github"},
			},
			wantError: false,
		},
		{
			name:  "wildcard mapping",
			input: "*:dgarifullin",
			want: map[string][]string{
				"*": {"dgarifullin"},
			},
			wantError: false,
		},
		{
			name:  "complex mapping",
			input: "alice:alice-github,alice:shared-github,bob:bob-github",
			want: map[string][]string{
				"alice": {"alice-github", "shared-github"},
				"bob":   {"bob-github"},
			},
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			want:      nil,
			wantError: true,
		},
		{
			name:      "invalid format - no colon",
			input:     "alice-alice-github",
			want:      nil,
			wantError: true,
		},
		{
			name:      "invalid format - empty SSH user",
			input:     ":alice-github",
			want:      nil,
			wantError: true,
		},
		{
			name:      "invalid format - empty GitHub user",
			input:     "alice:",
			want:      nil,
			wantError: true,
		},
		{
			name:      "invalid format - multiple colons",
			input:     "alice:github:extra",
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUserMap(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseUserMap() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if len(got) != len(tt.want) {
					t.Errorf("ParseUserMap() length = %d, want %d", len(got), len(tt.want))
					return
				}
				for k, v := range tt.want {
					if gotV, ok := got[k]; !ok {
						t.Errorf("ParseUserMap() missing key %q", k)
					} else {
						if len(gotV) != len(v) {
							t.Errorf("ParseUserMap() [%q] length = %d, want %d", k, len(gotV), len(v))
						}
						for i, wantVal := range v {
							if i >= len(gotV) || gotV[i] != wantVal {
								t.Errorf("ParseUserMap() [%q][%d] = %q, want %q", k, i, gotV[i], wantVal)
							}
						}
					}
				}
			}
		})
	}
}

func TestValidateLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"debug", "debug", false},
		{"info", "info", false},
		{"warn", "warn", false},
		{"error", "error", false},
		{"uppercase", "DEBUG", false},
		{"mixed case", "InFo", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLogLevel(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateLogLevel(%q) error = %v, wantError %v", tt.input, err, tt.wantError)
			}
		})
	}
}

func TestConfig_GetGitHubUsers(t *testing.T) {
	cfg := &Config{
		UserMap: map[string][]string{
			"alice": {"alice-github", "shared-github"},
			"bob":   {"bob-github"},
			"*":     {"wildcard-user"},
		},
	}

	tests := []struct {
		name         string
		sshUsername  string
		want         []string
		wantWildcard bool
	}{
		{"exact match", "alice", []string{"alice-github", "shared-github"}, false},
		{"exact match single", "bob", []string{"bob-github"}, false},
		{"wildcard match", "unknown", []string{"wildcard-user"}, true},
		{"wildcard match for nonexistent", "nonexistent", []string{"wildcard-user"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetGitHubUsers(tt.sshUsername)
			if len(got) != len(tt.want) {
				t.Errorf("GetGitHubUsers(%q) length = %d, want %d", tt.sshUsername, len(got), len(tt.want))
				return
			}
			for i, wantVal := range tt.want {
				if i >= len(got) || got[i] != wantVal {
					t.Errorf("GetGitHubUsers(%q)[%d] = %q, want %q", tt.sshUsername, i, got[i], wantVal)
				}
			}
		})
	}
}

