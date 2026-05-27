package operator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chaseputnam/ai-radio-fm/config"
)

type MockCM struct {
	mu            sync.Mutex
	talkGenCount  int
	musicGenCount int
	level         int
}

func (m *MockCM) GenerateTalk(ctx context.Context, show *config.Show) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.talkGenCount++
	return nil
}
func (m *MockCM) GenerateMusic(ctx context.Context, show *config.Show) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.musicGenCount++
	return nil
}
func (m *MockCM) InventoryLevel(showID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.level
}
func (m *MockCM) TalkGenCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.talkGenCount
}
func (m *MockCM) MusicGenCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.musicGenCount
}

type MockSM struct {
	mu           sync.Mutex
	healthy      bool
	restartCount int
}

func (m *MockSM) IsHealthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.healthy
}
func (m *MockSM) RestartStream() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.restartCount++
	m.healthy = true
	return nil
}
func (m *MockSM) RestartCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restartCount
}

func TestDaemon(t *testing.T) {
	cm := &MockCM{level: 2}
	sm := &MockSM{healthy: false}

	schedule := &config.ScheduleConfig{
		Shows: []config.Show{{ID: "show1"}},
	}

	d := NewDaemon(cm, sm, schedule, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d.Start(ctx, 10*time.Millisecond)

	time.Sleep(25 * time.Millisecond)
	d.Stop()

	if sm.RestartCount() == 0 {
		t.Error("expected stream restart")
	}
	if cm.TalkGenCount() == 0 {
		t.Error("expected talk generation")
	}
	if cm.MusicGenCount() == 0 {
		t.Error("expected music generation")
	}
}
