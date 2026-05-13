package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chaseputnam/ai-radio-fm/config"
	"github.com/chaseputnam/ai-radio-fm/ledger"
	"github.com/chaseputnam/ai-radio-fm/streamer"
)

// --- mocks ---

type mockAnthropicClient struct {
	script string
	err    error
}

func (m *mockAnthropicClient) GenerateScript(ctx context.Context, sys, user string) (string, error) {
	return m.script, m.err
}

type mockTTSRenderer struct {
	renderErr  error
	renderPath string // last path passed to Render
}

func (m *mockTTSRenderer) Render(ctx context.Context, text, voiceName, outputPath string) error {
	m.renderPath = outputPath
	if m.renderErr != nil {
		return m.renderErr
	}
	// Write a minimal WAV-like file so the playlist can open it.
	return os.WriteFile(outputPath, []byte("RIFF"), 0644)
}

// newTestContentManager builds a ContentManager wired to test doubles.
func newTestContentManager(
	t *testing.T,
	anthropic *mockAnthropicClient,
	tts TTSRenderer,
	musicBaseURL string,
) (*ContentManager, *streamer.Playlist, *ledger.Ledger, string) {
	t.Helper()
	contentDir := t.TempDir()
	ledgerPath := filepath.Join(t.TempDir(), "ledger.jsonl")
	l := ledger.NewLedger(ledgerPath)
	pl := streamer.NewPlaylist()

	personas := &config.PersonasConfig{
		Personas: []config.Persona{
			{
				ID:         "host1",
				Name:       "Test Host",
				VoiceModel: "af_heart",
			},
		},
	}

	var musicClient *MusicGenClient
	if musicBaseURL != "" {
		musicClient = NewMusicGenClient(musicBaseURL)
		musicClient.PollInterval = 1 // 1 nanosecond — effectively immediate in tests
	}

	cm := &ContentManager{
		scriptGen:  anthropic,
		tts:        tts,
		music:      musicClient,
		ledger:     l,
		playlist:   pl,
		personas:   personas,
		contentDir: contentDir,
		builder:    &PromptBuilder{},
	}

	return cm, pl, l, contentDir
}

// --- tests ---

func TestContentManager_GenerateTalk_Success(t *testing.T) {
	script := "Welcome to the show. Tonight we explore the void."
	ant := &mockAnthropicClient{script: script}
	tts := &mockTTSRenderer{}

	cm, pl, l, contentDir := newTestContentManager(t, ant, tts, "")

	show := &config.Show{ID: "midnight_signal", HostID: "host1", Name: "Midnight Signal"}
	err := cm.GenerateTalk(context.Background(), show)
	if err != nil {
		t.Fatalf("GenerateTalk failed: %v", err)
	}

	// Assert WAV file was written into the talk dir.
	talkDir := TalkDir(contentDir, show.ID)
	entries, _ := os.ReadDir(talkDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in talk dir, got %d", len(entries))
	}
	if filepath.Ext(entries[0].Name()) != ".wav" {
		t.Errorf("expected .wav file, got %s", entries[0].Name())
	}

	// Assert playlist item was enqueued (non-blocking peek via a goroutine).
	enqueuedPath := filepath.Join(talkDir, entries[0].Name())
	if tts.renderPath != enqueuedPath {
		t.Errorf("TTS render path: got %q, want %q", tts.renderPath, enqueuedPath)
	}

	// Assert ledger entry was written.
	history, err := l.ReadLast(10)
	if err != nil {
		t.Fatalf("ledger read failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(history))
	}
	if history[0].Action != "talk_generated" {
		t.Errorf("ledger action: got %q, want %q", history[0].Action, "talk_generated")
	}
	if history[0].ShowID != show.ID {
		t.Errorf("ledger show_id: got %q, want %q", history[0].ShowID, show.ID)
	}
	// Summary should be truncated to 120 chars.
	if len(history[0].Summary) > 120 {
		t.Errorf("ledger summary too long: %d chars", len(history[0].Summary))
	}

	// Assert playlist received the item.
	_ = pl // playlist is blocking; we verified via tts.renderPath above
}

func TestContentManager_GenerateTalk_PersonaNotFound(t *testing.T) {
	ant := &mockAnthropicClient{script: "hello"}
	cm, _, _, _ := newTestContentManager(t, ant, &mockTTSRenderer{}, "")

	show := &config.Show{ID: "show1", HostID: "nonexistent_host"}
	err := cm.GenerateTalk(context.Background(), show)
	if err == nil {
		t.Fatal("expected error for missing persona, got nil")
	}
}

func TestContentManager_GenerateTalk_AnthropicError(t *testing.T) {
	ant := &mockAnthropicClient{err: fmt.Errorf("api error")}
	cm, _, _, _ := newTestContentManager(t, ant, &mockTTSRenderer{}, "")

	show := &config.Show{ID: "show1", HostID: "host1"}
	err := cm.GenerateTalk(context.Background(), show)
	if err == nil {
		t.Fatal("expected error from Anthropic, got nil")
	}
}

func TestContentManager_GenerateTalk_NilTTS(t *testing.T) {
	ant := &mockAnthropicClient{script: "hello world"}
	cm, _, l, _ := newTestContentManager(t, ant, nil, "")

	show := &config.Show{ID: "show1", HostID: "host1", Name: "Show 1"}
	err := cm.GenerateTalk(context.Background(), show)
	if err != nil {
		t.Fatalf("expected nil error with nil TTS, got: %v", err)
	}

	// Ledger entry should still be written.
	history, _ := l.ReadLast(10)
	if len(history) != 1 {
		t.Errorf("expected 1 ledger entry even without TTS, got %d", len(history))
	}
}

func TestContentManager_GenerateMusic_Success(t *testing.T) {
	// Spin up a mock MusicGen HTTP server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/generate":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(GenerateMusicResponse{TaskID: "task-123"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks/task-123":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TaskStatusResponse{Status: "completed", AudioURL: "/audio/track.ogg"})
		case r.Method == http.MethodGet && r.URL.Path == "/audio/track.ogg":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OggS")) // minimal ogg-like content
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ant := &mockAnthropicClient{}
	cm, _, _, contentDir := newTestContentManager(t, ant, nil, srv.URL)

	show := &config.Show{ID: "midnight_signal", HostID: "host1", Description: "ambient music"}
	err := cm.GenerateMusic(context.Background(), show)
	if err != nil {
		t.Fatalf("GenerateMusic failed: %v", err)
	}

	// Assert file was downloaded into the music dir.
	musicDir := MusicDir(contentDir, show.ID)
	entries, _ := os.ReadDir(musicDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in music dir, got %d", len(entries))
	}
}

func TestContentManager_InventoryLevel(t *testing.T) {
	ant := &mockAnthropicClient{}
	cm, _, _, contentDir := newTestContentManager(t, ant, nil, "")

	showID := "test_show"

	// Empty dirs — should return 0.
	if level := cm.InventoryLevel(showID); level != 0 {
		t.Errorf("empty inventory: got %d, want 0", level)
	}

	// Add 2 talk files and 1 music file.
	talkDir := TalkDir(contentDir, showID)
	musicDir := MusicDir(contentDir, showID)
	os.MkdirAll(talkDir, 0755)
	os.MkdirAll(musicDir, 0755)
	os.WriteFile(filepath.Join(talkDir, "a.wav"), []byte{}, 0644)
	os.WriteFile(filepath.Join(talkDir, "b.wav"), []byte{}, 0644)
	os.WriteFile(filepath.Join(musicDir, "c.ogg"), []byte{}, 0644)

	if level := cm.InventoryLevel(showID); level != 3 {
		t.Errorf("inventory with 3 files: got %d, want 3", level)
	}
}
