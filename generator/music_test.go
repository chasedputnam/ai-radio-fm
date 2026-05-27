package generator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMusicGenClient_Generate_Success(t *testing.T) {
	// Known audio bytes to round-trip through base64.
	audioContent := []byte("fake-flac-audio-data")
	encoded := base64.StdEncoding.EncodeToString(audioContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/generate" {
			http.NotFound(w, r)
			return
		}
		// Verify request body fields.
		var req musicGenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Caption == "" {
			http.Error(w, "missing caption", http.StatusBadRequest)
			return
		}
		if !req.Instrumental {
			http.Error(w, "expected instrumental=true", http.StatusBadRequest)
			return
		}
		if req.Lyrics != "[Instrumental]" {
			http.Error(w, "expected [Instrumental] lyrics", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(musicGenResponse{
			Audios: []string{encoded},
			Metadata: struct {
				AudioFormat string `json:"audio_format"`
			}{AudioFormat: "flac"},
		})
	}))
	defer srv.Close()

	client := NewMusicGenClient(srv.URL, "flac")
	outDir := t.TempDir()

	filePath, err := client.Generate(context.Background(), "ambient lo-fi beats", outDir, 0)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Assert file exists with .flac extension.
	if filepath.Ext(filePath) != ".flac" {
		t.Errorf("expected .flac extension, got %q", filepath.Ext(filePath))
	}

	// Assert file is inside outDir.
	if dir := filepath.Dir(filePath); dir != outDir {
		t.Errorf("file %q not inside outDir %q", filePath, outDir)
	}

	// Assert file content matches original audio bytes.
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != string(audioContent) {
		t.Errorf("file content mismatch: got %q, want %q", content, audioContent)
	}
}

func TestMusicGenClient_Generate_EmptyAudios(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(musicGenResponse{Audios: []string{}})
	}))
	defer srv.Close()

	client := NewMusicGenClient(srv.URL, "flac")
	_, err := client.Generate(context.Background(), "test", t.TempDir(), 0)
	if err == nil {
		t.Fatal("expected error for empty audios, got nil")
	}
	if !strings.Contains(err.Error(), "no audio") {
		t.Errorf("expected 'no audio' in error, got: %v", err)
	}
}

func TestMusicGenClient_Generate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewMusicGenClient(srv.URL, "flac")
	_, err := client.Generate(context.Background(), "test", t.TempDir(), 0)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status 500 in error, got: %v", err)
	}
}

func TestMusicGenClient_Generate_FormatFallback(t *testing.T) {
	// When metadata.audio_format is empty, the file extension should fall back
	// to the client's configured AudioFormat.
	audioContent := []byte("fake-mp3-data")
	encoded := base64.StdEncoding.EncodeToString(audioContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(musicGenResponse{
			Audios: []string{encoded},
			Metadata: struct {
				AudioFormat string `json:"audio_format"`
			}{AudioFormat: ""}, // empty — should fall back to client format
		})
	}))
	defer srv.Close()

	client := NewMusicGenClient(srv.URL, "mp3")
	outDir := t.TempDir()

	filePath, err := client.Generate(context.Background(), "test", outDir, 0)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if filepath.Ext(filePath) != ".mp3" {
		t.Errorf("expected .mp3 extension from fallback, got %q", filepath.Ext(filePath))
	}
}
