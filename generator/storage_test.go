package generator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTalkDir(t *testing.T) {
	got := TalkDir("/data/content", "midnight_signal")
	want := filepath.Join("/data/content", "talk", "midnight_signal")
	if got != want {
		t.Errorf("TalkDir: got %q, want %q", got, want)
	}
}

func TestMusicDir(t *testing.T) {
	got := MusicDir("/data/content", "midnight_signal")
	want := filepath.Join("/data/content", "music", "midnight_signal")
	if got != want {
		t.Errorf("MusicDir: got %q, want %q", got, want)
	}
}

func TestCountAudioFiles_NonExistentDir(t *testing.T) {
	count, err := CountAudioFiles("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("expected nil error for missing dir, got: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for missing dir, got %d", count)
	}
}

func TestCountAudioFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	count, err := CountAudioFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for empty dir, got %d", count)
	}
}

func TestCountAudioFiles_MixedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create audio files
	for _, name := range []string{"a.wav", "b.wav", "c.ogg", "d.flac"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Create non-audio files that should not be counted
	for _, name := range []string{"notes.txt", "cover.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a subdirectory that should not be counted
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	count, err := CountAudioFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 audio files, got %d", count)
	}
}
