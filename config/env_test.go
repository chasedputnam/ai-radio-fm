package config

import (
	"testing"
)

func TestLoadEnv_Defaults(t *testing.T) {
	// Unset all relevant env vars to ensure defaults are used.
	vars := []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "ICECAST_HOST", "ICECAST_PORT", "ICECAST_MOUNT",
		"ICECAST_USER", "ICECAST_PASSWORD", "CONTENT_DIR", "KOKORO_LIB_PATH",
		"KOKORO_MODEL_PATH", "KOKORO_VOICE_DIR", "MUSICGEN_URL", "ARCHIVE_DIR",
		"LEDGER_PATH", "API_ADDR",
	}
	for _, v := range vars {
		t.Setenv(v, "")
	}

	cfg := LoadEnv()

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"AnthropicAPIKey", cfg.AnthropicAPIKey, ""},
		{"AnthropicBaseURL", cfg.AnthropicBaseURL, "https://api.anthropic.com"},
		{"IcecastHost", cfg.IcecastHost, "localhost"},
		{"IcecastMount", cfg.IcecastMount, "stream"},
		{"IcecastUser", cfg.IcecastUser, "source"},
		{"IcecastPassword", cfg.IcecastPassword, "hackme"},
		{"ContentDir", cfg.ContentDir, "./content"},
		{"KokoroLibPath", cfg.KokoroLibPath, "/opt/homebrew/lib/libonnxruntime.dylib"},
		{"KokoroModelPath", cfg.KokoroModelPath, "./go-kokoro-tts/kokoro-v0_19.onnx"},
		{"KokoroVoiceDir", cfg.KokoroVoiceDir, "./go-kokoro-tts/voices"},
		{"MusicGenURL", cfg.MusicGenURL, "http://localhost:8002"},
		{"TTSGRPCAddr", cfg.TTSGRPCAddr, ""},
		{"ArchiveDir", cfg.ArchiveDir, "./archive"},
		{"LedgerPath", cfg.LedgerPath, "./ledger.jsonl"},
		{"APIAddr", cfg.APIAddr, ":8001"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}

	if cfg.IcecastPort != 8000 {
		t.Errorf("IcecastPort: got %d, want 8000", cfg.IcecastPort)
	}
}

func TestLoadEnv_Overrides(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://openrouter.ai/api")
	t.Setenv("ICECAST_HOST", "radio.example.com")
	t.Setenv("ICECAST_PORT", "8080")
	t.Setenv("ICECAST_MOUNT", "live")
	t.Setenv("ICECAST_USER", "admin")
	t.Setenv("ICECAST_PASSWORD", "secret")
	t.Setenv("CONTENT_DIR", "/data/content")
	t.Setenv("KOKORO_LIB_PATH", "/usr/lib/libonnxruntime.so")
	t.Setenv("KOKORO_MODEL_PATH", "/models/kokoro.onnx")
	t.Setenv("KOKORO_VOICE_DIR", "/models/voices")
	t.Setenv("MUSICGEN_URL", "http://musicgen:9000")
	t.Setenv("TTS_GRPC_ADDR", "localhost:50051")
	t.Setenv("ARCHIVE_DIR", "/data/archive")
	t.Setenv("LEDGER_PATH", "/data/ledger.jsonl")
	t.Setenv("API_ADDR", ":9001")

	cfg := LoadEnv()

	if cfg.AnthropicAPIKey != "sk-test-key" {
		t.Errorf("AnthropicAPIKey: got %q, want %q", cfg.AnthropicAPIKey, "sk-test-key")
	}
	if cfg.AnthropicBaseURL != "https://openrouter.ai/api" {
		t.Errorf("AnthropicBaseURL: got %q", cfg.AnthropicBaseURL)
	}
	if cfg.IcecastHost != "radio.example.com" {
		t.Errorf("IcecastHost: got %q", cfg.IcecastHost)
	}
	if cfg.IcecastPort != 8080 {
		t.Errorf("IcecastPort: got %d, want 8080", cfg.IcecastPort)
	}
	if cfg.IcecastMount != "live" {
		t.Errorf("IcecastMount: got %q", cfg.IcecastMount)
	}
	if cfg.IcecastUser != "admin" {
		t.Errorf("IcecastUser: got %q", cfg.IcecastUser)
	}
	if cfg.IcecastPassword != "secret" {
		t.Errorf("IcecastPassword: got %q", cfg.IcecastPassword)
	}
	if cfg.ContentDir != "/data/content" {
		t.Errorf("ContentDir: got %q", cfg.ContentDir)
	}
	if cfg.KokoroLibPath != "/usr/lib/libonnxruntime.so" {
		t.Errorf("KokoroLibPath: got %q", cfg.KokoroLibPath)
	}
	if cfg.KokoroModelPath != "/models/kokoro.onnx" {
		t.Errorf("KokoroModelPath: got %q", cfg.KokoroModelPath)
	}
	if cfg.KokoroVoiceDir != "/models/voices" {
		t.Errorf("KokoroVoiceDir: got %q", cfg.KokoroVoiceDir)
	}
	if cfg.MusicGenURL != "http://musicgen:9000" {
		t.Errorf("MusicGenURL: got %q", cfg.MusicGenURL)
	}
	if cfg.TTSGRPCAddr != "localhost:50051" {
		t.Errorf("TTSGRPCAddr: got %q", cfg.TTSGRPCAddr)
	}
	if cfg.ArchiveDir != "/data/archive" {
		t.Errorf("ArchiveDir: got %q", cfg.ArchiveDir)
	}
	if cfg.LedgerPath != "/data/ledger.jsonl" {
		t.Errorf("LedgerPath: got %q", cfg.LedgerPath)
	}
	if cfg.APIAddr != ":9001" {
		t.Errorf("APIAddr: got %q", cfg.APIAddr)
	}
}

func TestLoadEnv_InvalidPort(t *testing.T) {
	t.Setenv("ICECAST_PORT", "not-a-number")
	cfg := LoadEnv()
	// Invalid port should fall back to default
	if cfg.IcecastPort != 8000 {
		t.Errorf("IcecastPort with invalid value: got %d, want 8000", cfg.IcecastPort)
	}
}
