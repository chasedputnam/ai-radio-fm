package generator

import (
	"os"
	"path/filepath"
)

// TalkDir returns the directory where talk segments for a show are stored.
func TalkDir(contentDir, showID string) string {
	return filepath.Join(contentDir, "talk", showID)
}

// MusicDir returns the directory where music tracks for a show are stored.
func MusicDir(contentDir, showID string) string {
	return filepath.Join(contentDir, "music", showID)
}

// CountAudioFiles returns the number of .wav and .ogg files in dir.
// Returns 0, nil if the directory does not exist.
func CountAudioFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext == ".wav" || ext == ".ogg" {
			count++
		}
	}
	return count, nil
}
