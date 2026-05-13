package config

import "path/filepath"

// StationPaths holds all filesystem paths for a named station.
// All paths are relative to the current working directory.
type StationPaths struct {
	Name         string
	Dir          string // stations/<name>/
	ScheduleFile string // stations/<name>/schedule.yaml
	PersonasFile string // stations/<name>/personas.yaml
	ContentDir   string // stations/<name>/content/
	ArchiveDir   string // stations/<name>/archive/
	LedgerPath   string // stations/<name>/ledger.jsonl
}

// StationPathsFor returns the canonical paths for a station with the given name.
func StationPathsFor(name string) StationPaths {
	dir := filepath.Join("stations", name)
	return StationPaths{
		Name:         name,
		Dir:          dir,
		ScheduleFile: filepath.Join(dir, "schedule.yaml"),
		PersonasFile: filepath.Join(dir, "personas.yaml"),
		ContentDir:   filepath.Join(dir, "content"),
		ArchiveDir:   filepath.Join(dir, "archive"),
		LedgerPath:   filepath.Join(dir, "ledger.jsonl"),
	}
}
