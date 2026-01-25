package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	manager, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	if manager.authorizedKeysPath == "" {
		t.Error("authorizedKeysPath is empty")
	}
}

func TestNewManagerWithPath(t *testing.T) {
	testPath := "/tmp/test-authorized_keys"
	manager := NewManagerWithPath(testPath)
	if manager == nil {
		t.Fatal("NewManagerWithPath() returned nil")
	}
	if manager.GetAuthorizedKeysPath() != testPath {
		t.Errorf("GetAuthorizedKeysPath() = %q, want %q", manager.GetAuthorizedKeysPath(), testPath)
	}
}

func TestManager_ReadExistingKeys(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		wantKeys    []string
		wantError   bool
	}{
		{
			name:        "empty file",
			fileContent: "",
			wantKeys:    []string{},
			wantError:   false,
		},
		{
			name: "single key",
			fileContent: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n",
			wantKeys:    []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com"},
			wantError:   false,
		},
		{
			name: "multiple keys",
			fileContent: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com
`,
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com",
			},
			wantError: false,
		},
		{
			name: "with comments",
			fileContent: `# This is a comment
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com
# Another comment
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com
`,
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com",
			},
			wantError: false,
		},
		{
			name: "with empty lines",
			fileContent: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com

ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com
`,
			wantKeys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			authKeysPath := filepath.Join(tmpDir, "authorized_keys")
			if err := os.WriteFile(authKeysPath, []byte(tt.fileContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			manager := NewManagerWithPath(authKeysPath)
			keys, err := manager.ReadExistingKeys()

			if (err != nil) != tt.wantError {
				t.Errorf("ReadExistingKeys() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("ReadExistingKeys() returned %d keys, want %d", len(keys), len(tt.wantKeys))
				return
			}

			for i, wantKey := range tt.wantKeys {
				if i >= len(keys) {
					t.Errorf("ReadExistingKeys() missing key at index %d: %q", i, wantKey)
					continue
				}
				if keys[i] != wantKey {
					t.Errorf("ReadExistingKeys()[%d] = %q, want %q", i, keys[i], wantKey)
				}
			}
		})
	}
}

func TestManager_ReadExistingKeys_FileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	authKeysPath := filepath.Join(tmpDir, "nonexistent_authorized_keys")

	manager := NewManagerWithPath(authKeysPath)
	keys, err := manager.ReadExistingKeys()

	if err != nil {
		t.Errorf("ReadExistingKeys() error = %v, want nil for non-existent file", err)
	}
	if len(keys) != 0 {
		t.Errorf("ReadExistingKeys() returned %d keys, want 0 for non-existent file", len(keys))
	}
}

func TestManager_MergeKeys(t *testing.T) {
	tests := []struct {
		name         string
		githubKeys   []string
		existingKeys []string
		wantCount    int
		wantContains []string
	}{
		{
			name:         "no existing keys",
			githubKeys:   []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB github@example.com"},
			existingKeys: []string{},
			wantCount:    1,
			wantContains: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB github@example.com"},
		},
		{
			name:         "no GitHub keys",
			githubKeys:   []string{},
			existingKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB existing@example.com"},
			wantCount:    1,
			wantContains: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB existing@example.com"},
		},
		{
			name:         "merge both",
			githubKeys:   []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB github@example.com"},
			existingKeys: []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI existing@example.com"},
			wantCount:    2,
			wantContains: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB github@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI existing@example.com",
			},
		},
		{
			name:         "deduplicate same key",
			githubKeys:   []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com"},
			existingKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com"},
			wantCount:    1,
			wantContains: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB same@example.com"},
		},
		{
			name:         "deduplicate different comments",
			githubKeys:   []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB github@example.com"},
			existingKeys: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB existing@example.com"},
			wantCount:    1, // Same key data, different comment - should deduplicate
			wantContains: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManagerWithPath("/tmp/test")
			merged := manager.MergeKeys(tt.githubKeys, tt.existingKeys)

			if len(merged) != tt.wantCount {
				t.Errorf("MergeKeys() returned %d keys, want %d", len(merged), tt.wantCount)
			}

			// Check that all expected keys are present
			mergedMap := make(map[string]bool)
			for _, key := range merged {
				mergedMap[key] = true
			}

			for _, wantKey := range tt.wantContains {
				found := false
				for _, mergedKey := range merged {
					if strings.Contains(mergedKey, wantKey) || strings.Contains(wantKey, mergedKey) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("MergeKeys() missing expected key containing: %q", wantKey)
				}
			}
		})
	}
}

func TestFormatKeys(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		want     string
		wantEnds bool
	}{
		{
			name:     "empty",
			keys:     []string{},
			want:     "",
			wantEnds: false,
		},
		{
			name:     "single key",
			keys:     []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com"},
			want:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com\n",
			wantEnds: true,
		},
		{
			name: "multiple keys",
			keys: []string{
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com",
				"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com",
			},
			want: `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB key1@example.com
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI key2@example.com
`,
			wantEnds: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatKeys(tt.keys)
			if got != tt.want {
				t.Errorf("FormatKeys() = %q, want %q", got, tt.want)
			}
			if tt.wantEnds && !strings.HasSuffix(got, "\n") {
				t.Error("FormatKeys() result should end with newline")
			}
		})
	}
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"simple", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB test@example.com", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB"},
		{"no comment", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB"},
		{"extra spaces", "ssh-rsa   AAAAB3NzaC1yc2EAAAADAQABAAAB   test@example.com", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB"},
		{"empty", "", ""},
		{"whitespace", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeKey(tt.key)
			if got != tt.want {
				t.Errorf("normalizeKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestManager_GetAllKeys(t *testing.T) {
	tmpDir := t.TempDir()
	authKeysPath := filepath.Join(tmpDir, "authorized_keys")

	// Create existing keys file
	existingContent := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB existing@example.com\n"
	if err := os.WriteFile(authKeysPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewManagerWithPath(authKeysPath)
	githubKeys := []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI github@example.com"}

	result, err := manager.GetAllKeys(githubKeys)
	if err != nil {
		t.Fatalf("GetAllKeys() error = %v", err)
	}

	if !strings.Contains(result, "existing@example.com") {
		t.Error("GetAllKeys() missing existing key")
	}
	if !strings.Contains(result, "github@example.com") {
		t.Error("GetAllKeys() missing GitHub key")
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("GetAllKeys() result should end with newline")
	}
}

