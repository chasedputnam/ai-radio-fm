package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Archiver struct {
	OutputDir string
	Playlist  *Playlist
	
	mu          sync.Mutex
	currentFile *os.File
	currentShow string
	fileStart   time.Time
}

func NewArchiver(outputDir string, playlist *Playlist) *Archiver {
	return &Archiver{
		OutputDir: outputDir,
		Playlist:  playlist,
	}
}

func (a *Archiver) Write(p []byte) (n int, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	show := a.Playlist.CurrentShow()
	now := time.Now()

	rollover := false
	if a.currentFile == nil {
		rollover = true
	} else if a.currentShow != show && show != "" {
		rollover = true
	} else if now.Sub(a.fileStart) > time.Hour {
		rollover = true
	}

	if rollover {
		if a.currentFile != nil {
			a.currentFile.Close()
		}

		if err := os.MkdirAll(a.OutputDir, 0755); err != nil {
			return 0, err
		}

		filename := fmt.Sprintf("archive_%s_%s.ogg", now.Format("2006-01-02T15-04-05"), show)
		if show == "" {
		    filename = fmt.Sprintf("archive_%s.ogg", now.Format("2006-01-02T15-04-05"))
		}
		f, err := os.Create(filepath.Join(a.OutputDir, filename))
		if err != nil {
			return 0, err
		}

		a.currentFile = f
		a.currentShow = show
		a.fileStart = now
	}

	return a.currentFile.Write(p)
}

func (a *Archiver) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.currentFile != nil {
		err := a.currentFile.Close()
		a.currentFile = nil
		return err
	}
	return nil
}
