package ledger

import (
	"os"
	"testing"
)

func TestLedger(t *testing.T) {
	tmpfile, _ := os.CreateTemp("", "ledger*.jsonl")
	defer os.Remove(tmpfile.Name())

	l := NewLedger(tmpfile.Name())

	entry1 := LedgerEntry{
		Action:  "generated_talk",
		ShowID:  "midnight_signal",
		Summary: "Talked about the moon",
	}

	err := l.Append(entry1)
	if err != nil {
		t.Fatal(err)
	}

	entry2 := LedgerEntry{
		Action:  "generated_music",
		ShowID:  "midnight_signal",
		Summary: "Generated upbeat jazz track",
	}

	err = l.Append(entry2)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := l.ReadLast(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Action != "generated_music" {
		t.Errorf("expected generated_music, got %s", entries[0].Action)
	}

	entries, err = l.ReadLast(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Action != "generated_talk" {
		t.Errorf("expected generated_talk, got %s", entries[0].Action)
	}
}
