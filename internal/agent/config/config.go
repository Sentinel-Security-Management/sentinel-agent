package config
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// K8sMetadata holds downward API injected values.
type K8sMetadata struct {
	NodeName      string
	PodName       string
	PodUID        string
	NamespaceName string
	ContainerName string
}

// Config represents the strictly typed configuration for sentinel-agent.
type Config struct {
	OTLPEndpoint    string
	OTLPTLSEnabled  bool
	OTLPCertFile    string
	OTLPKeyFile     string
	OTLPCAFile      string
	LogPaths        []string
	K8sEnabled      string
	DockerEnabled   string
	ScrubPolicyPath string
	MemoryLimitMiB  int
	QueueMaxSize    int
	LogLevel        string
	K8s             K8sMetadata
}

// Load reads all environment variables, applies defaults, and validates.
// It never panics, returning a descriptive error on validation failure.
func Load() (*Config, error) {
	cfg := &Config{
		OTLPEndpoint:    getEnv("SENTINEL_OTLP_ENDPOINT", "localhost:4317"),
		OTLPTLSEnabled:  getEnvAsBool("SENTINEL_OTLP_TLS", false),
		OTLPCertFile:    getEnv("SENTINEL_OTLP_CERT_FILE", ""),
		OTLPKeyFile:     getEnv("SENTINEL_OTLP_KEY_FILE", ""),
		OTLPCAFile:      getEnv("SENTINEL_OTLP_CA_FILE", ""),
		LogPaths:        getEnvAsSlice("SENTINEL_LOG_PATHS", "/var/log/*.log"),
		K8sEnabled:      getEnv("SENTINEL_K8S_ENABLED", "auto"),
		DockerEnabled:   getEnv("SENTINEL_DOCKER_ENABLED", "auto"),
		ScrubPolicyPath: getEnv("SENTINEL_SCRUB_POLICY", "/etc/sentinel/sentinel-policy.yaml"),
		MemoryLimitMiB:  getEnvAsInt("SENTINEL_MEMORY_LIMIT_MIB", 400),
		QueueMaxSize:    getEnvAsInt("SENTINEL_QUEUE_MAX_SIZE", 5000),
		LogLevel:        getEnv("SENTINEL_LOG_LEVEL", "info"),
		K8s: K8sMetadata{
			NodeName:      getEnv("NODE_NAME", ""),
			PodName:       getEnv("POD_NAME", ""),
			PodUID:        getEnv("POD_UID", ""),
			NamespaceName: getEnv("POD_NAMESPACE", ""),
			ContainerName: getEnv("CONTAINER_NAME", ""),
		},
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	// Validate Endpoint
	if _, _, err := net.SplitHostPort(cfg.OTLPEndpoint); err != nil {
		return fmt.Errorf("invalid SENTINEL_OTLP_ENDPOINT: must be a valid host:port string, got %q", cfg.OTLPEndpoint)
	}

	// Validate mTLS configuration
	if cfg.OTLPTLSEnabled {
		if cfg.OTLPCertFile == "" || cfg.OTLPKeyFile == "" {
			return fmt.Errorf("SENTINEL_OTLP_CERT_FILE and SENTINEL_OTLP_KEY_FILE must be set when SENTINEL_OTLP_TLS is true")
		}
		if _, err := os.Stat(cfg.OTLPCertFile); err != nil {
			return fmt.Errorf("cert file not found: %s", cfg.OTLPCertFile)
		}
		if _, err := os.Stat(cfg.OTLPKeyFile); err != nil {
			return fmt.Errorf("key file not found: %s", cfg.OTLPKeyFile)
		}
	}

	// Validate Log Level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid SENTINEL_LOG_LEVEL: must be debug, info, warn, or error, got %q", cfg.LogLevel)
	}

	// Validate feature flags
	validFlags := map[string]bool{"auto": true, "true": true, "false": true}
	if !validFlags[cfg.K8sEnabled] {
		return fmt.Errorf("invalid SENTINEL_K8S_ENABLED: must be auto, true, or false, got %q", cfg.K8sEnabled)
	}
	if !validFlags[cfg.DockerEnabled] {
		return fmt.Errorf("invalid SENTINEL_DOCKER_ENABLED: must be auto, true, or false, got %q", cfg.DockerEnabled)
	}

	// Validate limits
	if cfg.MemoryLimitMiB <= 100 {
		return fmt.Errorf("invalid SENTINEL_MEMORY_LIMIT_MIB: must be greater than 100, got %d", cfg.MemoryLimitMiB)
	}
	if cfg.QueueMaxSize <= 0 {
		return fmt.Errorf("invalid SENTINEL_QUEUE_MAX_SIZE: must be positive, got %d", cfg.QueueMaxSize)
	}

	return nil
}

// --- Helper Functions ---

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback // Fallback on parse error to ensure stability
	}
	return b
}

func getEnvAsInt(key string, fallback int) int {
	val := getEnv(key, "")
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvAsSlice(key string, fallback string) []string {
	val := getEnv(key, fallback)
	if val == "" {
		return nil
	}
	// Split by comma and trim spaces
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}