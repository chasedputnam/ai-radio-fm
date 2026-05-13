package config

import (
	"os"
	"testing"
)

func TestLoadSchedule(t *testing.T) {
	content := `shows:
  - id: midnight_signal
    name: "Midnight Signal"
    host_id: liminal_operator
    start_time: "00:00"
    end_time: "04:00"
    description: "Late night philosophy"`

	tmpfile, err := os.CreateTemp("", "schedule*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadSchedule(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadSchedule failed: %v", err)
	}

	if len(cfg.Shows) != 1 {
		t.Fatalf("expected 1 show, got %d", len(cfg.Shows))
	}
	if cfg.Shows[0].ID != "midnight_signal" {
		t.Errorf("expected id midnight_signal, got %s", cfg.Shows[0].ID)
	}
}

func TestLoadPersonas(t *testing.T) {
	content := `personas:
  - id: liminal_operator
    name: "The Liminal Operator"
    voice_model: "am_michael"
    philosophy: "Philosophy, radio lore"
    prompt_rules:
      - "Speak calmly"`

	tmpfile, err := os.CreateTemp("", "personas*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPersonas(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadPersonas failed: %v", err)
	}

	if len(cfg.Personas) != 1 {
		t.Fatalf("expected 1 persona, got %d", len(cfg.Personas))
	}
	if cfg.Personas[0].ID != "liminal_operator" {
		t.Errorf("expected id liminal_operator, got %s", cfg.Personas[0].ID)
	}
}
