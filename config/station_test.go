package config

import (
	"path/filepath"
	"testing"
)

func TestStationPathsFor(t *testing.T) {
	cases := []struct {
		name string
	}{
		{"midnight_fm"},
		{"the-night-garden"},
		{"station1"},
		{"my_station_2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := StationPathsFor(tc.name)

			if p.Name != tc.name {
				t.Errorf("Name: got %q, want %q", p.Name, tc.name)
			}

			wantDir := filepath.Join("stations", tc.name)
			if p.Dir != wantDir {
				t.Errorf("Dir: got %q, want %q", p.Dir, wantDir)
			}
			if p.ScheduleFile != filepath.Join(wantDir, "schedule.yaml") {
				t.Errorf("ScheduleFile: got %q", p.ScheduleFile)
			}
			if p.PersonasFile != filepath.Join(wantDir, "personas.yaml") {
				t.Errorf("PersonasFile: got %q", p.PersonasFile)
			}
			if p.ContentDir != filepath.Join(wantDir, "content") {
				t.Errorf("ContentDir: got %q", p.ContentDir)
			}
			if p.ArchiveDir != filepath.Join(wantDir, "archive") {
				t.Errorf("ArchiveDir: got %q", p.ArchiveDir)
			}
			if p.LedgerPath != filepath.Join(wantDir, "ledger.jsonl") {
				t.Errorf("LedgerPath: got %q", p.LedgerPath)
			}
		})
	}
}
