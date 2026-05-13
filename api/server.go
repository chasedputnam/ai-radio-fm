package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/chaseputnam/ai-radio-fm/config"
)

type NowPlayingInfo struct {
	ShowID   string `json:"show_id"`
	Track    string `json:"track"`
	HostName string `json:"host_name"`
}

type Server struct {
	addr         string
	schedule     *config.ScheduleConfig
	
	mu           sync.RWMutex
	nowPlaying   NowPlayingInfo
	healthStatus bool
}

func NewServer(addr string, schedule *config.ScheduleConfig) *Server {
	return &Server{
		addr:         addr,
		schedule:     schedule,
		healthStatus: true,
	}
}

func (s *Server) UpdateNowPlaying(info NowPlayingInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nowPlaying = info
}

func (s *Server) UpdateHealth(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthStatus = healthy
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/now-playing", s.handleNowPlaying)
	mux.HandleFunc("/schedule", s.handleSchedule)
	mux.HandleFunc("/health", s.handleHealth)

	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) handleNowPlaying(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	info := s.nowPlaying
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.schedule)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	healthy := s.healthStatus
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if healthy {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"error"}`))
	}
}
