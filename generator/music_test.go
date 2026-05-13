package generator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMusicGenClient(t *testing.T) {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(GenerateMusicResponse{TaskID: "task-123"})
	})
	
	pollCount := 0
	mux.HandleFunc("/api/tasks/task-123", func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		if pollCount < 2 {
			json.NewEncoder(w).Encode(TaskStatusResponse{Status: "processing"})
			return
		}
		json.NewEncoder(w).Encode(TaskStatusResponse{Status: "completed", AudioURL: "/download/track.ogg"})
	})

	mux.HandleFunc("/download/track.ogg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("audio data"))
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewMusicGenClient(server.URL)
	client.PollInterval = 10 * time.Millisecond
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	taskID, err := client.RequestGeneration(ctx, "upbeat jazz")
	if err != nil {
		t.Fatal(err)
	}
	if taskID != "task-123" {
		t.Errorf("expected task-123, got %s", taskID)
	}
	
	audioURL, err := client.WaitForCompletion(ctx, taskID)
	if err != nil {
		t.Fatal(err)
	}
	if audioURL != "/download/track.ogg" {
		t.Errorf("expected /download/track.ogg, got %s", audioURL)
	}

	tmpDir, _ := os.MkdirTemp("", "music*")
	defer os.RemoveAll(tmpDir)

	file, err := client.DownloadTrack(ctx, audioURL, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(file)
	if string(content) != "audio data" {
		t.Errorf("expected 'audio data', got %s", string(content))
	}
}
