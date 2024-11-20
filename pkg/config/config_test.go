package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type testDefaulter struct {
	called bool
	err    error
}

func (t *testDefaulter) setDefaults() error {
	t.called = true
	return t.err
}

type nestedStruct struct {
	Inner *testDefaulter
}

type mapStruct struct {
	Data map[string]string
}

type mapImplementer map[string]string

func (m *mapImplementer) setDefaults() error {
	(*m)["default"] = "set"
	return nil
}

func TestProcessValue(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		method  func(defaulter) error
		verify  func(t *testing.T, input interface{})
		wantErr bool
	}{
		{
			name:  "direct struct implementing interface",
			input: &testDefaulter{},
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			verify: func(t *testing.T, input interface{}) {
				d := input.(*testDefaulter)
				if !d.called {
					t.Error("method was not called on direct struct")
				}
			},
		},
		{
			name: "nested struct implementing interface",
			input: &nestedStruct{
				Inner: &testDefaulter{},
			},
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			verify: func(t *testing.T, input interface{}) {
				n := input.(*nestedStruct)
				if !n.Inner.called {
					t.Error("method was not called on nested struct")
				}
			},
		},
		{
			name: "error propagation",
			input: &testDefaulter{
				err: errors.New("test error"),
			},
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			wantErr: true,
		},
		{
			name:  "nil pointer",
			input: (*testDefaulter)(nil),
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			verify: func(t *testing.T, input interface{}) {
				// Should not panic and should not call method
			},
		},
		{
			name: "nil nested pointer",
			input: &nestedStruct{
				Inner: nil,
			},
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			verify: func(t *testing.T, input interface{}) {
				// Should not panic and should not call method
			},
		},
		{
			name: "map field",
			input: &mapStruct{
				Data: map[string]string{"key": "value"},
			},
			method: func(d defaulter) error {
				return nil
			},
			verify: func(t *testing.T, input interface{}) {
				// Should not panic when processing map field
			},
		},
		{
			name: "nil map field",
			input: &mapStruct{
				Data: nil,
			},
			method: func(d defaulter) error {
				return nil
			},
			verify: func(t *testing.T, input interface{}) {
				// Should not panic when processing nil map field
			},
		},
		{
			name:  "map implementing interface",
			input: &mapImplementer{"initial": "value"},
			method: func(d defaulter) error {
				return d.setDefaults()
			},
			verify: func(t *testing.T, input interface{}) {
				m := input.(*mapImplementer)
				if (*m)["default"] != "set" {
					t.Error("method was not called on map implementing interface")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processValue(reflect.ValueOf(tt.input), tt.method)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("processValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run verification if provided
			if tt.verify != nil {
				tt.verify(t, tt.input)
			}
		})
	}
}

type testGitExpected struct {
	remoteName string
	url        string
	credential string
	interval   time.Duration
}

func TestConfigGit(t *testing.T) {
	configDir := t.TempDir()

	tests := []struct {
		name     string
		input    string
		expected GitRepository
	}{
		{
			name: "default",
			input: `sources:
  git:
    default:
      remote:
        name: upstream
        url: https://corp-repos/default.git
  `,
			expected: GitRepository{
				Remote: &Remote{
					Name:     "upstream",
					URL:      "https://corp-repos/default.git",
					Interval: 10 * time.Second,
				},
				DefaultBranch: "main",
			},
		},
		{
			name: "custom",
			input: `sources:
  git:
    custom:
      remote:
        name: origin
        url: https://corp-repos/custom
        credential: vault
        interval: 1m
      path: v1
      default_branch: release-v1
  `,
			expected: GitRepository{
				Remote: &Remote{
					Name:       "origin",
					URL:        "https://corp-repos/custom",
					Credential: "vault",
					Interval:   time.Minute,
				},
				Path:          "v1",
				DefaultBranch: "release-v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(configDir, tt.name+"-glu.yaml")
			err := os.WriteFile(configPath, []byte(tt.input), 0600)
			if err != nil {
				t.Fatalf("failed to write configuration file: %v", err)
			}

			c, err := ReadFromPath(configPath)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			repos := c.Sources.Git
			if len(repos) == 0 {
				t.Fatalf("expected at least one repo, got zero")
			}
			repo, ok := repos[tt.name]
			if !ok {
				t.Fatalf("expected repo %s to exist", tt.name)
			}

			if !reflect.DeepEqual(tt.expected.Remote, repo.Remote) {
				t.Errorf("expected remote %v, got %v", tt.expected.Remote, repo.Remote)
			}
			if tt.expected.Path != repo.Path {
				t.Errorf("expected path %v, got %v", tt.expected.Path, repo.Path)
			}
			if tt.expected.DefaultBranch != repo.DefaultBranch {
				t.Errorf("expected default branch %v, got %v", tt.expected.DefaultBranch, repo.DefaultBranch)
			}
		})
	}
}

func TestReadFromFileWhenNoConfig(t *testing.T) {
	_, err := ReadFromPath(filepath.Join(t.TempDir(), "non-existent-file.yaml"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
