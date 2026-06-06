package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure environment is clean
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading defaults, got: %v", err)
	}

	if cfg.OTLPEndpoint != "localhost:4317" {
		t.Errorf("expected default endpoint localhost:4317, got: %s", cfg.OTLPEndpoint)
	}
	if cfg.MemoryLimitMiB != 400 {
		t.Errorf("expected default memory limit 400, got: %d", cfg.MemoryLimitMiB)
	}
	if cfg.K8sEnabled != "auto" {
		t.Errorf("expected K8sEnabled auto, got: %s", cfg.K8sEnabled)
	}
}

func TestValidationFailures(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name:    "Invalid Endpoint",
			envVars: map[string]string{"SENTINEL_OTLP_ENDPOINT": "http://localhost"}, // missing port
			wantErr: "invalid SENTINEL_OTLP_ENDPOINT",
		},
		{
			name:    "Invalid Memory Limit",
			envVars: map[string]string{"SENTINEL_MEMORY_LIMIT_MIB": "50"},
			wantErr: "invalid SENTINEL_MEMORY_LIMIT_MIB",
		},
		{
			name:    "Missing TLS Certs",
			envVars: map[string]string{"SENTINEL_OTLP_TLS": "true"},
			wantErr: "SENTINEL_OTLP_CERT_FILE and SENTINEL_OTLP_KEY_FILE must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			_, err := Load()
			if err == nil {
				t.Errorf("expected error containing %q, but got nil", tt.wantErr)
			}
		})
	}
}
