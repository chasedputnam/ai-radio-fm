package streamer

import (
	"io"
	"os"
	"sync"
)

type PlaylistItem struct {
	FilePath string
	ShowID   string
}

type Playlist struct {
	queue []PlaylistItem
	mu    sync.Mutex
	cond  *sync.Cond

	currentFile *os.File
	currentShow string

	// OnTrackChange is called (without the lock held) whenever the playlist
	// advances to a new file. It receives the PlaylistItem that just started.
	// The callback must not call any Playlist methods — the playlist lock is
	// re-acquired immediately after the callback returns, so any re-entrant
	// call to Enqueue, Read, or CurrentShow will deadlock.
	OnTrackChange func(item PlaylistItem)
}

func NewPlaylist() *Playlist {
	p := &Playlist{
		queue: make([]PlaylistItem, 0),
	}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *Playlist) Enqueue(item PlaylistItem) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append(p.queue, item)
	p.cond.Signal()
}

func (p *Playlist) Read(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		if p.currentFile == nil {
			for len(p.queue) == 0 {
				p.cond.Wait()
			}
			
			nextItem := p.queue[0]
			p.queue = p.queue[1:]

			f, err := os.Open(nextItem.FilePath)
			if err != nil {
				continue
			}
			p.currentFile = f
			p.currentShow = nextItem.ShowID

			// Capture callback and item before releasing the lock.
			cb := p.OnTrackChange
			item := nextItem
			if cb != nil {
				p.mu.Unlock()
				cb(item)
				p.mu.Lock()
			}
		}

		n, err := p.currentFile.Read(b)
		if n > 0 {
			return n, nil
		}

		if err == io.EOF {
			p.currentFile.Close()
			p.currentFile = nil
			continue
		}

		if err != nil {
			p.currentFile.Close()
			p.currentFile = nil
			return 0, err
		}
	}
}

func (p *Playlist) CurrentShow() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentShow
}
