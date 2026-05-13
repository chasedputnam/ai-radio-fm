package streamer

import (
	"bytes"
	"os"
	"testing"
)

func TestArchiver(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "archive*")
	defer os.RemoveAll(tmpDir)

	p := NewPlaylist()
	a := NewArchiver(tmpDir, p)
	
	f1, _ := os.CreateTemp("", "test*.ogg")
	f1.Write([]byte("data1"))
	f1.Close()
	defer os.Remove(f1.Name())

	p.Enqueue(PlaylistItem{FilePath: f1.Name(), ShowID: "show1"})

	buf := make([]byte, 5)
	p.Read(buf) 
	
	n, err := a.Write([]byte("test data"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 9 {
		t.Errorf("expected 9, got %d", n)
	}
	
	a.Close()

	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	
	content, _ := os.ReadFile(tmpDir + "/" + entries[0].Name())
	if !bytes.Equal(content, []byte("test data")) {
		t.Errorf("unexpected content: %s", string(content))
	}
}
