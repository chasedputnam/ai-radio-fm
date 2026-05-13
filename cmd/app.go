package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/chaseputnam/ai-radio-fm/api"
	"github.com/chaseputnam/ai-radio-fm/streamer"
)

// AppStreamMonitor implements operator.StreamMonitor against a live Icecast server.
type AppStreamMonitor struct {
	client     *streamer.Client
	icecastCfg streamer.IcecastConfig
	apiServer  *api.Server
	playlist   *streamer.Playlist
	archiver   *streamer.Archiver
	tee        io.Reader       // TeeReader(playlist, archiver), set by SetTee
	parentCtx  context.Context // root context; used to scope restarted stream goroutines

	mu           sync.Mutex
	streamCtx    context.Context
	cancelStream context.CancelFunc
}

// NewAppStreamMonitor creates an AppStreamMonitor. The provided parentCtx is
// the station root context; each stream goroutine gets its own child context
// so it can be cancelled independently on restart while still being bounded
// by the station lifetime.
func NewAppStreamMonitor(
	parentCtx context.Context,
	client *streamer.Client,
	cfg streamer.IcecastConfig,
	apiServer *api.Server,
	playlist *streamer.Playlist,
	archiver *streamer.Archiver,
) *AppStreamMonitor {
	streamCtx, cancel := context.WithCancel(parentCtx)
	return &AppStreamMonitor{
		client:       client,
		icecastCfg:   cfg,
		apiServer:    apiServer,
		playlist:     playlist,
		archiver:     archiver,
		parentCtx:    parentCtx,
		streamCtx:    streamCtx,
		cancelStream: cancel,
	}
}

// SetTee sets the io.Reader used by stream goroutines. Must be called before
// StartStream. Typically: io.TeeReader(playlist, archiver).
func (m *AppStreamMonitor) SetTee(r io.Reader) {
	m.mu.Lock()
	m.tee = r
	m.mu.Unlock()
}

// StartStream launches the initial stream goroutine. Call once after SetTee.
func (m *AppStreamMonitor) StartStream() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.launchStream()
}

// launchStream starts a new stream goroutine using the current streamCtx.
// Must be called with m.mu held.
func (m *AppStreamMonitor) launchStream() {
	r := m.tee
	if r == nil {
		r = io.TeeReader(m.playlist, m.archiver)
	}
	go func() {
		m.client.Stream(m.streamCtx, r) //nolint:errcheck // reconnect loop handles errors internally
	}()
}

// icecastStatusURL returns the Icecast status JSON endpoint URL.
func (m *AppStreamMonitor) icecastStatusURL() string {
	return fmt.Sprintf("http://%s:%d/status-json.xsl", m.icecastCfg.Host, m.icecastCfg.Port)
}

// IsHealthy queries the Icecast status endpoint and returns true only if the
// configured mount point is listed as an active source.
func (m *AppStreamMonitor) IsHealthy() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(m.icecastStatusURL())
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	// Icecast's status-json.xsl can return "source" as either an object or an
	// array depending on the number of active sources. We decode into a raw
	// message first and handle both shapes.
	var raw struct {
		Icestats struct {
			Source json.RawMessage `json:"source"`
		} `json:"icestats"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return false
	}

	if len(raw.Icestats.Source) == 0 {
		return false
	}

	mount := "/" + m.icecastCfg.Mount

	// Try array first.
	var sources []struct {
		Mount string `json:"mount"`
	}
	if err := json.Unmarshal(raw.Icestats.Source, &sources); err == nil {
		for _, s := range sources {
			if s.Mount == mount {
				return true
			}
		}
		return false
	}

	// Fall back to single object.
	var single struct {
		Mount string `json:"mount"`
	}
	if err := json.Unmarshal(raw.Icestats.Source, &single); err == nil {
		return single.Mount == mount
	}

	return false
}

// RestartStream cancels the current stream goroutine and starts a fresh one.
func (m *AppStreamMonitor) RestartStream() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Signal unhealthy while restarting.
	m.apiServer.UpdateHealth(false)

	// Cancel the existing stream goroutine.
	if m.cancelStream != nil {
		m.cancelStream()
	}

	// Create a new child context from the parent so the restarted stream is
	// cancelled when the station shuts down.
	m.streamCtx, m.cancelStream = context.WithCancel(m.parentCtx)
	m.launchStream()

	// Mark healthy again — the stream goroutine will reconnect with backoff.
	m.apiServer.UpdateHealth(true)

	return nil
}
