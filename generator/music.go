package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MusicGenClient calls the go-music-gen /generate endpoint synchronously.
// The server returns base64-encoded audio inline — no polling required.
type MusicGenClient struct {
	BaseURL     string
	AudioFormat string
	HTTPClient  *http.Client
}

// NewMusicGenClient creates a MusicGenClient targeting the given base URL.
// audioFormat is the requested output format (e.g. "flac", "mp3", "wav").
// The HTTP client timeout is 300 seconds to accommodate synchronous ACE-Step inference.
func NewMusicGenClient(baseURL, audioFormat string) *MusicGenClient {
	if audioFormat == "" {
		audioFormat = "flac"
	}
	return &MusicGenClient{
		BaseURL:     baseURL,
		AudioFormat: audioFormat,
		HTTPClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// musicGenRequest is the JSON body sent to POST /generate.
type musicGenRequest struct {
	Caption      string   `json:"caption"`
	Lyrics       string   `json:"lyrics"`
	Instrumental bool     `json:"instrumental"`
	AudioFormat  string   `json:"audio_format"`
	Duration     *float64 `json:"duration,omitempty"`
}

// musicGenResponse is the JSON body returned by POST /generate.
type musicGenResponse struct {
	Audios   []string `json:"audios"`
	Metadata struct {
		AudioFormat string `json:"audio_format"`
	} `json:"metadata"`
}

// Generate sends a POST /generate request to go-music-gen, decodes the first
// returned audio file from base64, writes it to outputDir with a timestamp
// filename, and returns the full file path.
// duration is the requested track length in seconds; 0 means use the server default.
func (c *MusicGenClient) Generate(ctx context.Context, caption, outputDir string, duration float64) (string, error) {
	reqBody := musicGenRequest{
		Caption:      caption,
		Lyrics:       "[Instrumental]",
		Instrumental: true,
		AudioFormat:  c.AudioFormat,
	}
	if duration > 0 {
		reqBody.Duration = &duration
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal music gen request: %w", err)
	}

	// The HTTP client has a 300-second timeout to accommodate synchronous
	// ACE-Step inference (60–180s on CPU for a 30-second track).
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimSuffix(c.BaseURL, "/")+"/generate", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create music gen request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("music gen request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("music gen returned status %d: %s", resp.StatusCode, string(body))
	}

	var genResp musicGenResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode music gen response: %w", err)
	}

	if len(genResp.Audios) == 0 {
		return "", fmt.Errorf("music gen returned no audio")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(genResp.Audios[0])
	if err != nil {
		return "", fmt.Errorf("failed to base64-decode audio: %w", err)
	}

	// Determine extension from response metadata, fall back to configured format.
	ext := genResp.Metadata.AudioFormat
	if ext == "" {
		ext = c.AudioFormat
	}
	ext = strings.TrimPrefix(ext, ".")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create music output dir: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102T150405.000Z")
	filePath := filepath.Join(outputDir, timestamp+"."+ext)

	if err := os.WriteFile(filePath, audioBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write music file: %w", err)
	}

	return filePath, nil
}
