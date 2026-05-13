package streamer

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestPlaylist(t *testing.T) {
	p := NewPlaylist()

	// Create dummy files
	f1, _ := os.CreateTemp("", "test1*.ogg")
	f1.Write([]byte("file1data"))
	f1.Close()
	defer os.Remove(f1.Name())

	f2, _ := os.CreateTemp("", "test2*.ogg")
	f2.Write([]byte("file2data"))
	f2.Close()
	defer os.Remove(f2.Name())

	p.Enqueue(PlaylistItem{FilePath: f1.Name(), ShowID: "show1"})
	p.Enqueue(PlaylistItem{FilePath: f2.Name(), ShowID: "show2"})

	buf := make([]byte, 20)
	
	// Read from file 1
	n, err := p.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf[:n], []byte("file1data")) {
		t.Errorf("expected file1data, got %s", string(buf[:n]))
	}

	if p.CurrentShow() != "show1" {
		t.Errorf("expected show1, got %s", p.CurrentShow())
	}

	// Read from file 2
	n, err = p.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf[:n], []byte("file2data")) {
		t.Errorf("expected file2data, got %s", string(buf[:n]))
	}

	if p.CurrentShow() != "show2" {
		t.Errorf("expected show2, got %s", p.CurrentShow())
	}
	
	// Test block wait
	done := make(chan bool)
	go func() {
		p.Read(buf)
		done <- true
	}()

	select {
	case <-done:
		t.Error("Read should have blocked")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

func TestPlaylist_OnTrackChange(t *testing.T) {
	p := NewPlaylist()

	// Create two temp files with distinct content.
	f1, _ := os.CreateTemp("", "track1*.ogg")
	f1.Write([]byte("track1"))
	f1.Close()
	defer os.Remove(f1.Name())

	f2, _ := os.CreateTemp("", "track2*.ogg")
	f2.Write([]byte("track2"))
	f2.Close()
	defer os.Remove(f2.Name())

	item1 := PlaylistItem{FilePath: f1.Name(), ShowID: "showA"}
	item2 := PlaylistItem{FilePath: f2.Name(), ShowID: "showB"}

	var received []PlaylistItem
	p.OnTrackChange = func(item PlaylistItem) {
		received = append(received, item)
	}

	p.Enqueue(item1)
	p.Enqueue(item2)

	buf := make([]byte, 64)

	// First read — opens file 1, fires callback.
	p.Read(buf)
	// Second read — exhausts file 1 (EOF), opens file 2, fires callback.
	p.Read(buf)

	if len(received) != 2 {
		t.Fatalf("expected 2 OnTrackChange calls, got %d", len(received))
	}
	if received[0].ShowID != "showA" || received[0].FilePath != f1.Name() {
		t.Errorf("first callback: got %+v, want showA / %s", received[0], f1.Name())
	}
	if received[1].ShowID != "showB" || received[1].FilePath != f2.Name() {
		t.Errorf("second callback: got %+v, want showB / %s", received[1], f2.Name())
	}
}
