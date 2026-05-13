package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chaseputnam/ai-radio-fm/config"
)

func TestServer(t *testing.T) {
	schedule := &config.ScheduleConfig{
		Shows: []config.Show{{ID: "show1", Name: "Show 1"}},
	}
	s := NewServer(":8080", schedule)
	
	s.UpdateNowPlaying(NowPlayingInfo{ShowID: "show1", Track: "track1.ogg", HostName: "Nyx"})
	s.UpdateHealth(true)

	req, _ := http.NewRequest("GET", "/now-playing", nil)
	rr := httptest.NewRecorder()
	
	s.handleNowPlaying(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
	
	var info NowPlayingInfo
	json.NewDecoder(rr.Body).Decode(&info)
	if info.Track != "track1.ogg" {
		t.Errorf("expected track1.ogg, got %s", info.Track)
	}

	req, _ = http.NewRequest("GET", "/health", nil)
	rr = httptest.NewRecorder()
	s.handleHealth(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	req, _ = http.NewRequest("GET", "/schedule", nil)
	rr = httptest.NewRecorder()
	s.handleSchedule(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
	
	var sched config.ScheduleConfig
	json.NewDecoder(rr.Body).Decode(&sched)
	if len(sched.Shows) != 1 {
		t.Errorf("expected 1 show, got %d", len(sched.Shows))
	}
}
