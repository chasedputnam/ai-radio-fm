# Requirements Document

## Introduction

AI Radio FM is a 24/7 autonomous internet radio station written in Go. The codebase has a well-defined architecture but several critical subsystems are stubbed out or unconnected, meaning the station cannot actually run end-to-end. This spec covers completing all deferred implementations so the binary operates as described in the README: loading real config, generating talk scripts via Anthropic, rendering them to audio via Kokoro TTS, queuing music from the MusicGen server, streaming everything to Icecast, archiving the broadcast, and exposing live status via the API.

The stubs and gaps identified in the audit are:

1. `cmd/app.go` — `AppContentManager.GenerateTalk`, `GenerateMusic`, and `InventoryLevel` are all stubs returning zero values
2. `cmd/app.go` — `AppStreamMonitor.IsHealthy` always returns `true`; `RestartStream` is a no-op
3. `cmd/start.go` — schedule is hardcoded inline; config files are never loaded; `AppContentManager` has no dependencies injected; `KokoroTTS`, `Ledger`, `Archiver`, and `apiServer.UpdateNowPlaying` are never wired together
4. `cmd/generate.go` — uses a hardcoded mock API key and hardcoded persona/show instead of loading from config
5. `streamer/client.go` — `Stream` takes an `io.Reader` but `cmd/start.go` passes a `*Playlist` directly; the archiver is constructed but never wired into the stream path
6. `generator/tts.go` — `EngineOptions` are not sourced from config; output directory for rendered audio is not configurable
7. No content storage layer — generated WAV files and music tracks have no defined output directory; the playlist has no way to discover what files exist on disk
8. `api/server.go` — `UpdateNowPlaying` is never called; now-playing always returns empty JSON

---

## Requirements

### Requirement 1 — Configuration Loading

**User Story:** As an operator, I want the station to load its schedule and personas from YAML files at startup, so that I can configure shows and hosts without recompiling.

#### Acceptance Criteria

- WHEN `airadio start` is invoked THEN the system SHALL load `config/schedule.yaml` and `config/personas.yaml` using the existing `config.LoadSchedule` and `config.LoadPersonas` functions
- IF either config file is missing or malformed THEN the system SHALL print a descriptive error and exit with a non-zero status code
- WHEN config is loaded THEN the `ScheduleConfig` and `PersonasConfig` SHALL be passed to all subsystems that require them (daemon, API server, content manager)
- WHEN `airadio generate talk` is invoked THEN the command SHALL load the same config files and use the first show and its associated persona rather than hardcoded values

---

### Requirement 2 — Content Storage Layout

**User Story:** As an operator, I want generated audio files to be stored in a predictable directory structure, so that the playlist can find and queue them reliably.

#### Acceptance Criteria

- WHEN the system starts THEN it SHALL use a configurable base content directory (default: `./content`)
- WHEN a talk segment is rendered THEN the WAV file SHALL be written to `{contentDir}/talk/{showID}/` with a timestamp-based filename
- WHEN a music track is downloaded THEN it SHALL be saved to `{contentDir}/music/{showID}/`
- WHEN the playlist queries inventory for a show THEN it SHALL count files present in the show's talk and music subdirectories
- IF the content directory does not exist THEN the system SHALL create it on startup

---

### Requirement 3 — AppContentManager Implementation

**User Story:** As an operator, I want the daemon's content manager to actually generate talk scripts and music when inventory is low, so that the station never runs out of content.

#### Acceptance Criteria

- WHEN `GenerateTalk` is called for a show THEN the system SHALL look up the show's `host_id` in the loaded personas config
- WHEN a matching persona is found THEN the system SHALL call `AnthropicClient.GenerateScript` with prompts built by `PromptBuilder`, using the Anthropic API key from the environment variable `ANTHROPIC_API_KEY`
- WHEN a script is returned THEN the system SHALL call `KokoroTTS.Render` using the persona's `voice_model` field as the voice name
- WHEN the WAV file is written THEN the system SHALL append a `LedgerEntry` recording the action, show ID, and a truncated summary of the script
- WHEN `GenerateMusic` is called THEN the system SHALL call `MusicGenClient.RequestGeneration` followed by `WaitForCompletion` and `DownloadTrack`, saving the result to the music content directory
- WHEN `InventoryLevel` is called THEN it SHALL return the actual count of audio files present in the show's content directory
- IF `ANTHROPIC_API_KEY` is not set THEN `GenerateTalk` SHALL return an error immediately
- IF the MusicGen server is unreachable THEN `GenerateMusic` SHALL return an error without crashing the daemon

---

### Requirement 4 — Streaming Pipeline Wiring

**User Story:** As a listener, I want the station to stream a continuous mix of talk and music to Icecast, so that I can tune in at any time.

#### Acceptance Criteria

- WHEN `airadio start` runs THEN the `Playlist` SHALL be passed as the `io.Reader` to `streamer.Client.Stream`
- WHEN a new talk or music file is generated THEN it SHALL be enqueued onto the `Playlist` automatically
- WHEN the `Playlist` is read THEN the data SHALL also be written to the `Archiver` so the broadcast is recorded to disk
- WHEN the stream disconnects from Icecast THEN the client SHALL reconnect with exponential backoff as already implemented
- WHEN a file is dequeued from the playlist THEN `api.Server.UpdateNowPlaying` SHALL be called with the current show ID and track filename

---

### Requirement 5 — AppStreamMonitor Implementation

**User Story:** As an operator, I want the daemon to detect and recover from stream failures, so that the station self-heals without manual intervention.

#### Acceptance Criteria

- WHEN `IsHealthy` is called THEN it SHALL query the Icecast server's `/status-json.xsl` endpoint (or equivalent) and return `true` only if the mount point is active
- IF the health check HTTP request fails or returns a non-200 status THEN `IsHealthy` SHALL return `false`
- WHEN `RestartStream` is called THEN it SHALL cancel the current stream context and start a new stream goroutine
- WHEN the stream is restarted THEN `api.Server.UpdateHealth` SHALL be called with `false` during restart and `true` once reconnected

---

### Requirement 6 — Environment and Startup Configuration

**User Story:** As an operator, I want to configure runtime parameters (API keys, paths, ports) via environment variables, so that I can deploy the station without modifying code.

#### Acceptance Criteria

- WHEN the system starts THEN it SHALL read the following environment variables: `ANTHROPIC_API_KEY`, `ICECAST_HOST`, `ICECAST_PORT`, `ICECAST_MOUNT`, `ICECAST_USER`, `ICECAST_PASSWORD`, `CONTENT_DIR`, `KOKORO_LIB_PATH`, `KOKORO_MODEL_PATH`, `KOKORO_VOICE_DIR`, `MUSICGEN_URL`
- WHEN an environment variable is not set THEN the system SHALL fall back to the default values documented in the README
- IF `KOKORO_LIB_PATH` or `KOKORO_MODEL_PATH` point to files that do not exist THEN the system SHALL log a warning and disable TTS generation rather than crashing

---

### Requirement 7 — Now Playing Updates

**User Story:** As a listener, I want the `/now-playing` API endpoint to reflect what is actually playing, so that clients and displays show accurate metadata.

#### Acceptance Criteria

- WHEN the playlist advances to a new file THEN `api.Server.UpdateNowPlaying` SHALL be called with the track filename and show ID
- WHEN `/now-playing` is queried THEN it SHALL return the most recently set `NowPlayingInfo`
- WHEN no track has played yet THEN `/now-playing` SHALL return a JSON object with empty string fields rather than a null body
