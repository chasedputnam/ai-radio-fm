package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/chaseputnam/ai-radio-fm/config"
)

type ContentManager interface {
	GenerateTalk(ctx context.Context, show *config.Show) error
	GenerateMusic(ctx context.Context, show *config.Show) error
	InventoryLevel(showID string) int
}

type StreamMonitor interface {
	IsHealthy() bool
	RestartStream() error
}

type Daemon struct {
	contentManager     ContentManager
	streamMonitor      StreamMonitor
	schedule           *config.ScheduleConfig
	ticker             *time.Ticker
	inventoryThreshold int
}

func NewDaemon(cm ContentManager, sm StreamMonitor, schedule *config.ScheduleConfig, inventoryThreshold int) *Daemon {
	if inventoryThreshold <= 0 {
		inventoryThreshold = 5
	}
	return &Daemon{
		contentManager:     cm,
		streamMonitor:      sm,
		schedule:           schedule,
		inventoryThreshold: inventoryThreshold,
	}
}

func (d *Daemon) Start(ctx context.Context, interval time.Duration) {
	d.ticker = time.NewTicker(interval)
	go d.loop(ctx)
}

func (d *Daemon) Stop() {
	if d.ticker != nil {
		d.ticker.Stop()
	}
}

func (d *Daemon) loop(ctx context.Context) {
	d.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Daemon) tick(ctx context.Context) {
	if !d.streamMonitor.IsHealthy() {
		fmt.Println("Operator: Stream is unhealthy. Restarting...")
		d.streamMonitor.RestartStream()
	}

	if len(d.schedule.Shows) > 0 {
		show := &d.schedule.Shows[0]
		level := d.contentManager.InventoryLevel(show.ID)
		
		if level < d.inventoryThreshold {
			fmt.Printf("Operator: Inventory low for %s. Generating content...\n", show.ID)
			err := d.contentManager.GenerateTalk(ctx, show)
			if err != nil {
				fmt.Printf("Operator: Failed to generate talk: %v\n", err)
			}
			
			err = d.contentManager.GenerateMusic(ctx, show)
			if err != nil {
				fmt.Printf("Operator: Failed to generate music: %v\n", err)
			}
		}
	}
}
