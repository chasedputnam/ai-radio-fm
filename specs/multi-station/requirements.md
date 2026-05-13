# Requirements Document

## Introduction

AI Radio FM currently runs as a single station. This feature enables up to five independent stations to run simultaneously on one machine, each with its own schedule, personas, content, and Icecast mount point. Shared infrastructure (Icecast, MusicGen server, a gRPC TTS sidecar) is deployed once and used by all stations. Each station runs as a separate OS process for crash isolation. An orchestration shell script manages starting, stopping, and monitoring all stations together.

The work spans two repositories:
- `ai-radio-fm` — station directory structure, per-station config, env var conventions, and the orchestration script
- `go-kokoro-tts` — a new `cmd/tts-server` gRPC sidecar that replaces the in-process ONNX engine for all stations

---

## Requirements

### Requirement 1 — Station Directory Structure

**User Story:** As an operator, I want each station's configuration and data isolated in its own directory, so that I can add, remove, or modify one station without touching any other.

#### Acceptance Criteria

- WHEN the operator creates a new station THEN they SHALL place `schedule.yaml` and `personas.yaml` under `stations/<station_name>/`
- WHEN a station process starts THEN it SHALL resolve all paths (content, archive, ledger) relative to `stations/<station_name>/` by default
- WHEN a station process starts THEN it SHALL read its schedule from `stations/<station_name>/schedule.yaml` and its personas from `stations/<station_name>/personas.yaml`
- IF `stations/<station_name>/schedule.yaml` or `stations/<station_name>/personas.yaml` is missing THEN the station process SHALL exit with a descriptive error
- WHEN a station runs THEN its content directory SHALL be `stations/<station_name>/content/`, archive SHALL be `stations/<station_name>/archive/`, and ledger SHALL be `stations/<station_name>/ledger.jsonl`
- WHEN the binary is invoked THEN it SHALL accept a `--station <name>` flag that sets the station name and resolves all paths accordingly

---

### Requirement 2 — Per-Station Icecast Mount Points

**User Story:** As an operator, I want each station to stream to its own Icecast mount point on a shared Icecast instance, so that listeners can tune into individual stations without port conflicts.

#### Acceptance Criteria

- WHEN a station starts THEN its Icecast mount point SHALL default to `/<station_name>` if `ICECAST_MOUNT` is not set
- WHEN two stations are running THEN they SHALL stream to different mount points on the same Icecast host and port
- WHEN the operator sets `ICECAST_MOUNT` in a station's environment THEN that value SHALL override the default
- IF two stations are configured with the same mount point THEN the orchestration script SHALL detect this and refuse to start, printing a conflict error
- WHEN `IsHealthy` checks the Icecast status THEN it SHALL check for the station's specific mount point, not any mount

---

### Requirement 3 — Per-Station API Server Ports

**User Story:** As an operator, I want each station's API server to listen on a unique port, so that I can query any station's status independently without conflicts.

#### Acceptance Criteria

- WHEN a station starts THEN its API server SHALL listen on the port defined by `API_ADDR` in its environment
- WHEN the orchestration script starts multiple stations THEN it SHALL assign a unique `API_ADDR` to each station
- IF two stations are configured with the same `API_ADDR` THEN the orchestration script SHALL detect this and refuse to start
- WHEN a station's API server fails to bind its port THEN the station process SHALL exit with a descriptive error rather than silently continuing

---

### Requirement 4 — gRPC TTS Sidecar

**User Story:** As an operator, I want a single TTS process shared by all stations, so that I don't load five separate ONNX engines into memory simultaneously.

#### Acceptance Criteria

- WHEN the TTS sidecar starts THEN it SHALL load the Kokoro ONNX model and voice profiles once and serve synthesis requests over gRPC
- WHEN a station calls the TTS sidecar THEN it SHALL send the text and voice name over gRPC and receive raw PCM float32 audio samples in return
- WHEN multiple stations call the TTS sidecar concurrently THEN it SHALL serialize requests through the existing `ONNXEngine` mutex and return correct results to each caller
- WHEN the TTS sidecar is unavailable THEN the station SHALL fall back to script-only mode (no audio rendered), log a warning, and continue operating
- WHEN the TTS sidecar starts THEN it SHALL listen on a configurable address (default `localhost:50051`) set via `--addr` flag or `TTS_GRPC_ADDR` environment variable
- WHEN a station starts THEN it SHALL connect to the TTS sidecar address defined by `TTS_GRPC_ADDR` (default `localhost:50051`) instead of loading a local ONNX engine
- IF the gRPC connection to the TTS sidecar fails at startup THEN the station SHALL log a warning and continue in script-only mode
- WHEN the TTS sidecar receives a synthesis request THEN it SHALL respond within 60 seconds or return a timeout error to the caller

---

### Requirement 5 — Shared MusicGen Server

**User Story:** As an operator, I want all stations to share a single MusicGen server, so that I only run one instance of that service.

#### Acceptance Criteria

- WHEN multiple stations are running THEN they SHALL all point to the same `MUSICGEN_URL`
- WHEN the orchestration script starts THEN it SHALL use a single `MUSICGEN_URL` value shared across all station environments
- IF the MusicGen server is unavailable THEN each station SHALL handle the error independently without affecting other stations

---

### Requirement 6 — Orchestration Script

**User Story:** As an operator, I want a single script to start, stop, and check the status of all stations and shared services, so that I can manage the full platform without running commands manually for each station.

#### Acceptance Criteria

- WHEN the operator runs `./stations.sh start` THEN the script SHALL start the TTS sidecar, then start each station defined in `stations/` as a separate background process
- WHEN the operator runs `./stations.sh stop` THEN the script SHALL send SIGTERM to all station processes and the TTS sidecar, waiting for each to exit cleanly
- WHEN the operator runs `./stations.sh status` THEN the script SHALL print the running/stopped state of the TTS sidecar and each station process
- WHEN the operator runs `./stations.sh restart <station_name>` THEN the script SHALL stop and restart only that station's process without affecting others
- WHEN the operator runs `./stations.sh logs <station_name>` THEN the script SHALL tail the log file for that station
- WHEN the script starts a station THEN it SHALL write the station process PID to `stations/<station_name>/station.pid`
- WHEN the script starts the TTS sidecar THEN it SHALL write its PID to `stations/tts-server.pid`
- IF a station's mount point or API port conflicts with another station THEN the script SHALL print a descriptive error and exit without starting any processes
- WHEN a station process exits unexpectedly THEN the script SHALL NOT automatically restart it — the operator must restart manually (no daemon supervision in v1)
- WHEN the script starts stations THEN it SHALL write each station's stdout and stderr to `stations/<station_name>/station.log`

---

### Requirement 7 — Station Name as CLI Flag

**User Story:** As an operator, I want to start a specific station by name from the CLI, so that the orchestration script can launch each station as an independent process with the correct configuration.

#### Acceptance Criteria

- WHEN `airadio start --station <name>` is invoked THEN the binary SHALL load config from `stations/<name>/` and use `<name>` as the default Icecast mount
- WHEN `--station` is not provided THEN the binary SHALL fall back to the existing single-station behaviour (loading from `config/`) for backwards compatibility
- IF `--station <name>` is provided and `stations/<name>/` does not exist THEN the binary SHALL exit with a descriptive error

---

### Requirement 8 — Example Station Configs

**User Story:** As an operator, I want example station directories I can copy and customise, so that I can set up a new station quickly.

#### Acceptance Criteria

- WHEN the repository is cloned THEN it SHALL contain at least one example station under `stations/example_station/` with a `schedule.yaml` and `personas.yaml`
- WHEN the operator copies `stations/example_station/` to a new name and edits the YAML files THEN the station SHALL start without any other changes required
