package ledger

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
	"time"
)

type LedgerEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	ShowID    string    `json:"show_id"`
	Summary   string    `json:"summary"`
	Tags      []string  `json:"tags"`
}

type Ledger struct {
	filePath string
	mu       sync.Mutex
}

func NewLedger(filePath string) *Ledger {
	return &Ledger{
		filePath: filePath,
	}
}

func (l *Ledger) Append(entry LedgerEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = f.Write(data)
	return err
}

func (l *Ledger) ReadLast(n int) ([]LedgerEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	f, err := os.Open(l.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []LedgerEntry{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var allEntries []LedgerEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry LedgerEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		allEntries = append(allEntries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(allEntries) > n {
		return allEntries[len(allEntries)-n:], nil
	}
	return allEntries, nil
}
