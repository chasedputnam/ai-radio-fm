package operator

import (
	"context"
	"testing"
	"time"

	"github.com/chaseputnam/ai-radio-fm/config"
)

type MockCM struct {
	talkGenCount  int
	musicGenCount int
	level         int
}

func (m *MockCM) GenerateTalk(ctx context.Context, show *config.Show) error {
	m.talkGenCount++
	return nil
}
func (m *MockCM) GenerateMusic(ctx context.Context, show *config.Show) error {
	m.musicGenCount++
	return nil
}
func (m *MockCM) InventoryLevel(showID string) int {
	return m.level
}

type MockSM struct {
	healthy      bool
	restartCount int
}

func (m *MockSM) IsHealthy() bool {
	return m.healthy
}
func (m *MockSM) RestartStream() error {
	m.restartCount++
	m.healthy = true
	return nil
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
	
	if sm.restartCount == 0 {
		t.Error("expected stream restart")
	}
	if cm.talkGenCount == 0 {
		t.Error("expected talk generation")
	}
	if cm.musicGenCount == 0 {
		t.Error("expected music generation")
	}
}
