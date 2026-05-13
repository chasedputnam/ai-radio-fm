package config

import (
	"os"
	"strconv"
)

// RuntimeConfig holds all runtime configuration sourced from environment variables.
// Defaults match the values documented in the README.
type RuntimeConfig struct {
	AnthropicAPIKey    string
	AnthropicBaseURL   string
	IcecastHost        string
	IcecastPort        int
	IcecastMount       string
	IcecastUser        string
	IcecastPassword    string
	ContentDir         string
	KokoroLibPath      string
	KokoroModelPath    string
	KokoroVoiceDir     string
	KokoroCoreML       bool
	MusicGenURL        string
	TTSGRPCAddr        string
	ArchiveDir         string
	LedgerPath         string
	APIAddr            string
	InventoryThreshold int
}

// LoadEnv reads runtime configuration from environment variables, falling back
// to sensible defaults when a variable is not set.
func LoadEnv() RuntimeConfig {
	return RuntimeConfig{
		AnthropicAPIKey:    getEnv("ANTHROPIC_API_KEY", ""),
		AnthropicBaseURL:   getEnv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
		IcecastHost:        getEnv("ICECAST_HOST", "localhost"),
		IcecastPort:        getEnvInt("ICECAST_PORT", 8000),
		IcecastMount:       getEnv("ICECAST_MOUNT", "stream"),
		IcecastUser:        getEnv("ICECAST_USER", "source"),
		IcecastPassword:    getEnv("ICECAST_PASSWORD", "hackme"),
		ContentDir:         getEnv("CONTENT_DIR", "./content"),
		KokoroLibPath:      getEnv("KOKORO_LIB_PATH", "/opt/homebrew/lib/libonnxruntime.dylib"),
		KokoroModelPath:    getEnv("KOKORO_MODEL_PATH", "./go-kokoro-tts/kokoro-v0_19.onnx"),
		KokoroVoiceDir:     getEnv("KOKORO_VOICE_DIR", "./go-kokoro-tts/voices"),
		KokoroCoreML:       getEnvBool("KOKORO_COREML", true),
		MusicGenURL:        getEnv("MUSICGEN_URL", "http://localhost:8002"),
		TTSGRPCAddr:        getEnv("TTS_GRPC_ADDR", ""),
		ArchiveDir:         getEnv("ARCHIVE_DIR", "./archive"),
		LedgerPath:         getEnv("LEDGER_PATH", "./ledger.jsonl"),
		APIAddr:            getEnv("API_ADDR", ":8001"),
		InventoryThreshold: getEnvInt("INVENTORY_THRESHOLD", 5),
	}
}

// getEnv returns the value of the environment variable named by key.
// Returns defaultVal if the variable is unset or empty.
func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// getEnvBool returns the boolean value of an environment variable.
// Accepts "true", "1", "yes" (case-insensitive) as true; anything else as false.
// Returns defaultVal if the variable is unset or empty.
func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch v {
	case "true", "1", "yes", "TRUE", "YES":
		return true
	default:
		return false
	}
}
