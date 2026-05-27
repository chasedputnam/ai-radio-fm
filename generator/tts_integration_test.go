package generator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasedputnam/go-kokoro-tts/pkg/model"
)

func TestKokoroTTSIntegration(t *testing.T) {
	// Only run this test if the ONNX library and model files exist locally.
	// This acts as a true integration test.
	libPath := "/opt/homebrew/lib/libonnxruntime.dylib"
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: %s not found", libPath)
	}

	modelPath := "../go-kokoro-tts/kokoro-v0_19.onnx"
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: %s not found", modelPath)
	}

	voiceDir := "../go-kokoro-tts/voices"
	voiceName := "af_heart"
	
	opts := model.EngineOptions{UseCoreML: true}
	tts, err := NewKokoroTTS(libPath, modelPath, "", voiceDir, opts)
	if err != nil {
		t.Fatalf("Failed to initialize TTS engine: %v", err)
	}
	defer tts.Close()

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "output.wav")

	err = tts.Render(context.Background(), "Hello, integration test.", voiceName, outPath)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	stat, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	if stat.Size() == 0 {
		t.Errorf("Output file is empty")
	}
}
