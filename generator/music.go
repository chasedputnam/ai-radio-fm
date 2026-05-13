package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MusicGenClient struct {
	BaseURL      string
	HTTPClient   *http.Client
	PollInterval time.Duration
}

type GenerateMusicRequest struct {
	Prompt string `json:"prompt"`
}

type GenerateMusicResponse struct {
	TaskID string `json:"task_id"`
}

type TaskStatusResponse struct {
	Status   string `json:"status"` // "pending", "processing", "completed", "failed"
	AudioURL string `json:"audio_url,omitempty"`
}

func NewMusicGenClient(baseURL string) *MusicGenClient {
	return &MusicGenClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		PollInterval: 5 * time.Second,
	}
}

func (c *MusicGenClient) RequestGeneration(ctx context.Context, prompt string) (string, error) {
	reqBody := GenerateMusicRequest{Prompt: prompt}
	data, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/generate", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var genResp GenerateMusicResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", err
	}

	return genResp.TaskID, nil
}

func (c *MusicGenClient) WaitForCompletion(ctx context.Context, taskID string) (string, error) {
	ticker := time.NewTicker(c.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tasks/%s", c.BaseURL, taskID), nil)
			if err != nil {
				return "", err
			}

			resp, err := c.HTTPClient.Do(req)
			if err != nil {
				continue 
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				continue
			}

			var statusResp TaskStatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if statusResp.Status == "completed" {
				return statusResp.AudioURL, nil
			} else if statusResp.Status == "failed" {
				return "", fmt.Errorf("music generation task failed")
			}
		}
	}
}

func (c *MusicGenClient) DownloadTrack(ctx context.Context, url, outputDir string) (string, error) {
	if strings.HasPrefix(url, "/") {
		url = c.BaseURL + url
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download track, status: %d", resp.StatusCode)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}

	filename := filepath.Join(outputDir, filepath.Base(url))
	if filepath.Ext(filename) == "" {
		filename += ".ogg"
	}
	
	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}

	return filename, nil
}
