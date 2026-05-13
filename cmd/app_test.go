package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chaseputnam/ai-radio-fm/api"
	"github.com/chaseputnam/ai-radio-fm/config"
	"github.com/chaseputnam/ai-radio-fm/streamer"
)

func newTestMonitor(t *testing.T, handler http.HandlerFunc) (*AppStreamMonitor, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	cfg := streamer.IcecastConfig{
		Host:     "127.0.0.1",
		Port:     testServerPort(srv),
		Mount:    "stream",
		User:     "source",
		Password: "hackme",
	}

	schedule := &config.ScheduleConfig{}
	apiSrv := api.NewServer(":0", schedule)

	monitor := &AppStreamMonitor{
		client:     streamer.NewClient(cfg),
		icecastCfg: cfg,
		apiServer:  apiSrv,
	}
	return monitor, srv
}

func testServerPort(srv *httptest.Server) int {
	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)
	return port
}

func TestAppStreamMonitor_IsHealthy_ActiveMount(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"icestats": map[string]interface{}{
				"source": []map[string]interface{}{
					{"mount": "/stream", "listenurl": "http://localhost:8000/stream"},
				},
			},
		})
	})

	monitor, _ := newTestMonitor(t, handler)
	if !monitor.IsHealthy() {
		t.Error("expected IsHealthy=true when mount is active")
	}
}

func TestAppStreamMonitor_IsHealthy_SingleSourceObject(t *testing.T) {
	// Icecast returns a single object (not array) when only one source is active.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"icestats": map[string]interface{}{
				"source": map[string]interface{}{
					"mount": "/stream",
				},
			},
		})
	})

	monitor, _ := newTestMonitor(t, handler)
	if !monitor.IsHealthy() {
		t.Error("expected IsHealthy=true for single-source object response")
	}
}

func TestAppStreamMonitor_IsHealthy_WrongMount(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"icestats": map[string]interface{}{
				"source": []map[string]interface{}{
					{"mount": "/other"},
				},
			},
		})
	})

	monitor, _ := newTestMonitor(t, handler)
	if monitor.IsHealthy() {
		t.Error("expected IsHealthy=false when mount does not match")
	}
}

func TestAppStreamMonitor_IsHealthy_NoSources(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"icestats": map[string]interface{}{},
		})
	})

	monitor, _ := newTestMonitor(t, handler)
	if monitor.IsHealthy() {
		t.Error("expected IsHealthy=false when no sources present")
	}
}

func TestAppStreamMonitor_IsHealthy_ServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	monitor, _ := newTestMonitor(t, handler)
	if monitor.IsHealthy() {
		t.Error("expected IsHealthy=false on server error")
	}
}

func TestAppStreamMonitor_IsHealthy_Unreachable(t *testing.T) {
	cfg := streamer.IcecastConfig{
		Host:  "127.0.0.1",
		Port:  19999, // nothing listening here
		Mount: "stream",
	}
	schedule := &config.ScheduleConfig{}
	apiSrv := api.NewServer(":0", schedule)
	monitor := &AppStreamMonitor{
		client:     streamer.NewClient(cfg),
		icecastCfg: cfg,
		apiServer:  apiSrv,
	}
	if monitor.IsHealthy() {
		t.Error("expected IsHealthy=false when server is unreachable")
	}
}
