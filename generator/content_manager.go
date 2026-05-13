package generator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chaseputnam/ai-radio-fm/config"
	"github.com/chaseputnam/ai-radio-fm/ledger"
	"github.com/chaseputnam/ai-radio-fm/streamer"
)

// scriptGenerator is the interface satisfied by *AnthropicClient and test mocks.
type scriptGenerator interface {
	GenerateScript(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// TTSRenderer is the interface satisfied by *KokoroTTS and test mocks.
type TTSRenderer interface {
	Render(ctx context.Context, text, voiceName, outputPath string) error
}

// ContentManager implements operator.ContentManager, wiring together the
// Anthropic script generator, Kokoro TTS renderer, MusicGen client, ledger,
// and playlist into a single content production pipeline.
type ContentManager struct {
	scriptGen  scriptGenerator
	tts        TTSRenderer // nil when Kokoro paths are unavailable
	music      *MusicGenClient
	ledger     *ledger.Ledger
	playlist   *streamer.Playlist
	personas   *config.PersonasConfig
	contentDir string
	builder    *PromptBuilder
}

// NewContentManager creates a fully wired ContentManager.
// tts may be nil; in that case talk generation skips audio rendering.
func NewContentManager(
	anthropic *AnthropicClient,
	tts TTSRenderer,
	music *MusicGenClient,
	l *ledger.Ledger,
	playlist *streamer.Playlist,
	personas *config.PersonasConfig,
	contentDir string,
) *ContentManager {
	return &ContentManager{
		scriptGen:  anthropic,
		tts:        tts,
		music:      music,
		ledger:     l,
		playlist:   playlist,
		personas:   personas,
		contentDir: contentDir,
		builder:    &PromptBuilder{},
	}
}

// findPersona looks up a persona by ID in the loaded personas config.
func (cm *ContentManager) findPersona(hostID string) (*config.Persona, error) {
	for i := range cm.personas.Personas {
		if cm.personas.Personas[i].ID == hostID {
			return &cm.personas.Personas[i], nil
		}
	}
	return nil, fmt.Errorf("persona not found for host_id %q", hostID)
}

// GenerateTalk generates a talk script for the show, renders it to audio via
// TTS, writes a ledger entry, and enqueues the resulting WAV onto the playlist.
func (cm *ContentManager) GenerateTalk(ctx context.Context, show *config.Show) error {
	persona, err := cm.findPersona(show.HostID)
	if err != nil {
		return fmt.Errorf("GenerateTalk: %w", err)
	}

	// Read recent ledger history for prompt context.
	history, err := cm.ledger.ReadLast(5)
	if err != nil {
		// Non-fatal: proceed without history.
		log.Printf("ContentManager: failed to read ledger history: %v", err)
		history = nil
	}
	summaries := make([]string, 0, len(history))
	for _, e := range history {
		summaries = append(summaries, e.Summary)
	}

	sysPrompt := cm.builder.BuildSystemPrompt(persona)
	userPrompt := cm.builder.BuildUserPrompt(show, summaries)

	script, err := cm.scriptGen.GenerateScript(ctx, sysPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("GenerateTalk: script generation failed: %w", err)
	}

	if cm.tts == nil {
		log.Printf("ContentManager: TTS unavailable, skipping audio render for show %q", show.ID)
	} else {
		// Build output path and ensure the directory exists.
		talkDir := TalkDir(cm.contentDir, show.ID)
		if err := os.MkdirAll(talkDir, 0755); err != nil {
			return fmt.Errorf("GenerateTalk: failed to create talk dir: %w", err)
		}
		timestamp := time.Now().UTC().Format("20060102T150405.000Z")
		outputPath := filepath.Join(talkDir, timestamp+".wav")

		if err := cm.tts.Render(ctx, script, persona.VoiceModel, outputPath); err != nil {
			return fmt.Errorf("GenerateTalk: TTS render failed: %w", err)
		}

		// Enqueue the rendered file onto the playlist.
		cm.playlist.Enqueue(streamer.PlaylistItem{
			FilePath: outputPath,
			ShowID:   show.ID,
		})
	}

	// Append ledger entry — non-fatal on failure.
	summary := script
	if runes := []rune(summary); len(runes) > 120 {
		summary = string(runes[:120])
	}
	summary = strings.ReplaceAll(summary, "\n", " ")
	if err := cm.ledger.Append(ledger.LedgerEntry{
		Action:  "talk_generated",
		ShowID:  show.ID,
		Summary: summary,
	}); err != nil {
		log.Printf("ContentManager: failed to write ledger entry: %v", err)
	}

	return nil
}

// GenerateMusic requests a music track from the MusicGen server, waits for
// completion, downloads the file, and enqueues it onto the playlist.
func (cm *ContentManager) GenerateMusic(ctx context.Context, show *config.Show) error {
	prompt := show.Description
	if prompt == "" {
		prompt = "ambient electronic music"
	}

	taskID, err := cm.music.RequestGeneration(ctx, prompt)
	if err != nil {
		return fmt.Errorf("GenerateMusic: request failed: %w", err)
	}

	audioURL, err := cm.music.WaitForCompletion(ctx, taskID)
	if err != nil {
		return fmt.Errorf("GenerateMusic: wait failed: %w", err)
	}

	musicDir := MusicDir(cm.contentDir, show.ID)
	if err := os.MkdirAll(musicDir, 0755); err != nil {
		return fmt.Errorf("GenerateMusic: failed to create music dir: %w", err)
	}

	filePath, err := cm.music.DownloadTrack(ctx, audioURL, musicDir)
	if err != nil {
		return fmt.Errorf("GenerateMusic: download failed: %w", err)
	}

	cm.playlist.Enqueue(streamer.PlaylistItem{
		FilePath: filePath,
		ShowID:   show.ID,
	})

	return nil
}

// InventoryLevel returns the total number of audio files available for a show.
// Returns 0 on any filesystem error so the daemon treats it as low inventory.
func (cm *ContentManager) InventoryLevel(showID string) int {
	talkCount, err := CountAudioFiles(TalkDir(cm.contentDir, showID))
	if err != nil {
		return 0
	}
	musicCount, err := CountAudioFiles(MusicDir(cm.contentDir, showID))
	if err != nil {
		return 0
	}
	return talkCount + musicCount
}
