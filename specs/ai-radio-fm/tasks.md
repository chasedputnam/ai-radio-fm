# Implementation Tasks

- [x] 1. Core CLI and Project Scaffolding
  - Setup the Go module and base CLI structure using `cobra`.
  - Parse YAML configuration files for the weekly schedule and host personas.
  - References: Requirement 3, Requirement 5
  - [x] 1.1 Initialize `go mod` and install Cobra dependency.
  - [x] 1.2 Create `cmd/root.go`, `cmd/start.go`, and `cmd/generate.go`.
  - [x] 1.3 Implement YAML parsing for `config/schedule.yaml` mapping shows and times.
  - [x] 1.4 Implement YAML parsing for `config/personas.yaml` mapping hosts and voices.

- [x] 2. Streaming Engine (Icecast Source Client & Playlist Management)
  - Implement the audio orchestration that pushes Ogg Vorbis data to Icecast.
  - References: Requirement 1
  - [x] 2.1 Implement an HTTP PUT client connecting to Icecast with basic auth and Ogg Vorbis headers.
  - [x] 2.2 Implement a playlist queue struct that seamlessly reads Ogg files into the Icecast stream.
  - [x] 2.3 Implement the show boundary logic: load appropriate bumpers and intros when a new show begins.
  - [x] 2.4 Add disconnect/reconnect logic with exponential backoff for the Icecast client.

- [x] 3. Stream Archival
  - Intercept the continuous audio stream and persist it to disk in chunks.
  - References: Requirement 6
  - [x] 3.1 Create an archiver struct that accepts an `io.Reader` and writes to an `io.Writer` on the filesystem.
  - [x] 3.2 Wire the archiver to tee off the data being sent from the Streaming Engine to Icecast.
  - [x] 3.3 Implement the rollover logic to create a new file and update metadata when the show boundary changes or file duration hits 1 hour.

- [x] 4. Content Generation Manager (Talk & TTS)
  - Manage the programmatic generation of spoken content using Anthropic API and Kokoro.
  - References: Requirement 2, Requirement 3
  - [x] 4.1 Implement Anthropic API client in Go.
  - [x] 4.2 Create a prompt builder that combines host persona, active show topic, and ledger history.
  - [x] 4.3 Execute Anthropic API call to generate the text script.
  - [x] 4.4 Implement `os/exec` wrapper to run Kokoro TTS Python script (or call its local API) to render the script text to an audio file.

- [x] 5. Content Generation Manager (Music)
  - Request AI music tracks from the external `music-gen.server`.
  - References: Requirement 2
  - [x] 5.1 Implement HTTP client for `music-gen.server` REST API.
  - [x] 5.2 Implement a polling or webhook mechanism to wait for the completed music tracks.
  - [x] 5.3 Fetch and store the completed `.ogg` music files into the local content pool for the streaming engine.

- [x] 6. Station Ledger
  - Maintain the editorial memory of the station.
  - References: Requirement 4
  - [x] 6.1 Implement an append-only JSONL logger for the station ledger.
  - [x] 6.2 Implement a reader that fetches the last N entries to pass to the Content Generation prompt builder.

- [x] 7. Operator Daemon
  - Create the background loop that orchestrates content levels and health.
  - References: Requirement 4
  - [x] 7.1 Implement the background goroutine loop ticking every 15 minutes.
  - [x] 7.2 Implement content inventory checks for the next 2-4 hours of programming.
  - [x] 7.3 Trigger the Content Generation Manager if inventory is low.
  - [x] 7.4 Monitor health of the stream/Icecast and trigger self-healing (restarts) if needed.

- [x] 8. API Server Integration
  - Expose station status, schedule, and now-playing metadata.
  - References: Requirement 5
  - [x] 8.1 Initialize an HTTP API server using `net/http` running concurrently on a separate port.
  - [x] 8.2 Add thread-safe access (mutexes/channels) for the Streaming Engine to report the currently playing track.
  - [x] 8.3 Implement `GET /now-playing` endpoint.
  - [x] 8.4 Implement `GET /schedule` endpoint.
  - [x] 8.5 Implement `GET /health` endpoint for the Operator Daemon and external monitoring.