package config

import (
	"errors"
	"log/slog"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	os.Exit(m.Run())
}

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

func TestConfig(t *testing.T) {
	tests := []struct {
		path     string
		expected *Config
	}{
		{
			path: "testdata/json",
			expected: &Config{
				Log: Log{Level: "info"},
				Server: Server{
					Port:     8080,
					Host:     "0.0.0.0",
					Protocol: "http",
				},
				Metrics: Metrics{
					Enabled:  true,
					Exporter: MetricsExporterPrometheus,
				},
				Storage: Storage{
					Type: "clickhouse",
					ClickHouse: &StorageClickHouse{
						Address:  "127.0.0.1:9000",
						Database: "default",
						Username: "default",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			c, err := ReadFromFS(os.DirFS(tt.path))
			require.NoError(t, err)

			assert.Equal(t, tt.expected, c)
		})
	}
}

func TestReadFromFileWhenNoConfig(t *testing.T) {
	_, err := ReadFromFS(os.DirFS("."))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
