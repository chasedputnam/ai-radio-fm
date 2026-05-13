# Design Document

## 1. Overview
The AI Radio FM system is a Golang-based rewrite of the existing Python/bash/tmux station. The goal is to provide a single, compiled binary that orchestrates the entire station: streaming audio, serving the API, running the autonomous operator loop, and executing content generation tasks. The Go application will interface with external services (Icecast for broadcasting, Anthropic API for script generation, Kokoro TTS for voice rendering, and music-gen for music creation).

## 2. Architecture
The system follows a concurrent, event-driven architecture utilizing Go routines to manage parallel domains. 

```mermaid
graph TD
    subgraph Go Application "AI Radio FM (Go Binary)"
        CLI[CLI Commands]
        API[HTTP API Server]
        Operator[Operator Daemon Loop]
        Streamer[Streaming Engine / Playlist Manager]
        Gen[Content Generation Manager]
        Archiver[Stream Archiver]
    end
    
    subgraph External Services
        Icecast[Icecast Server]
        Kokoro[Kokoro TTS Subprocess/API]
        Claude[Anthropic API / Claude]
        MusicGen[music-gen.server]
    end

    CLI -->|start/stop/generate| API
    Operator -->|health check / trigger gen| Gen
    Streamer -->|audio stream| Icecast
    Streamer -->|audio stream| Archiver
    Archiver -->|save segment| Disk[(Local Disk)]
    API -->|status queries| Streamer
    Gen -->|text generation| Claude
    Gen -->|audio render| Kokoro
    Gen -->|music tracks| MusicGen
    Operator -->|read/write| Ledger[(Station Ledger)]
```

## 3. Components and Interfaces

### 3.1. CLI and API
- **CLI (`cmd/airadio`):** Built using `spf13/cobra`. Provides commands like `start`, `generate talk`, `generate music`, and `status`.
- **API Server:** Built using standard `net/http` or `gin-gonic/gin`. Runs in a goroutine when the station starts.
  - `GET /now-playing`: Returns the current track, show, and host.
  - `GET /schedule`: Returns upcoming shows.
  - `GET /health`: Returns system health.

### 3.2. Streaming Engine & Archiver
- **Playlist Manager:** Monitors a target duration of queued content based on the current schedule. It stitches together talk blocks and music bumpers.
- **Icecast Source Client:** Instead of relying on `ezstream`, the Go app will pipe Ogg Vorbis files sequentially into an HTTP PUT request to the Icecast server, acting as the source client. Alternatively, it can manage an `ffmpeg` subprocess to continuously transcode and stream a named pipe or HTTP stream.
- **Stream Archiver:** A component attached to the Streaming Engine output. It intercepts the Ogg Vorbis data being sent to Icecast and writes it to a local file. It implements a rollover mechanism to close the current file and open a new one when a configured time interval (e.g., hourly) or a show boundary is reached.

### 3.3. Content Generation Manager
- Orchestrates the creation of new content.
- **Talk Generation:** Calls the Anthropic API (or Claude CLI) using the persona prompts and station ledger to generate scripts.
- **TTS Rendering:** Executes the local Kokoro TTS engine (via `os/exec` subprocess or HTTP API) to render scripts into `.ogg`/`.wav` files.
- **Music Generation:** Makes HTTP REST calls to `music-gen.server`.

### 3.4. Operator Daemon
- A background goroutine that ticks on a schedule (e.g., every 15 minutes).
- Checks content inventory for the next 2-4 hours.
- Triggers the Content Generation Manager if inventory is low.
- Appends editorial decisions to the append-only Station Ledger (`ledger.jsonl`).

## 4. Data Models

### 4.1. Schedule & Persona
```go
type Show struct {
    ID          string   `yaml:"id"`
    Name        string   `yaml:"name"`
    HostID      string   `yaml:"host_id"`
    StartTime   string   `yaml:"start_time"` // "HH:MM"
    EndTime     string   `yaml:"end_time"`
    Description string   `yaml:"description"`
}

type Persona struct {
    ID          string   `yaml:"id"`
    Name        string   `yaml:"name"`
    VoiceModel  string   `yaml:"voice_model"`
    Philosophy  string   `yaml:"philosophy"`
    PromptRules []string `yaml:"prompt_rules"`
}
```

### 4.2. Station Ledger Entry
```go
type LedgerEntry struct {
    Timestamp time.Time `json:"timestamp"`
    Action    string    `json:"action"`   // e.g., "generated_talk", "listener_response"
    ShowID    string    `json:"show_id"`
    Summary   string    `json:"summary"`
    Tags      []string  `json:"tags"`
}
```

## 5. Error Handling
- **Stream Interruptions:** The Streaming Engine will implement exponential backoff and retry logic if the Icecast server disconnects. It will retain the current playlist state to resume playback smoothly.
- **Generation Failures:** If an API call to Claude or Kokoro fails, the operator loop will log the error and retry on the next tick. The streaming engine will fall back to a pool of "evergreen" generic content if the scheduled content buffer runs entirely empty.
- **Concurrency:** Uses Go channels and `sync.RWMutex` to safely update the `/now-playing` state between the streaming engine and the API server.

## 6. Testing Strategy
- **Unit Tests:** For schedule parsing, playlist logic, and ledger appending.
- **Integration Tests:** For the API endpoints and the Content Generation Manager (using mocked Anthropic and music-gen API responses).
- **Stream Mocking:** A mock Icecast server (simple HTTP server accepting PUT requests) to verify the Streaming Engine behaves correctly under disconnect/reconnect scenarios.

**Risk:** Direct Ogg Vorbis streaming via Go to Icecast without `ezstream`/`ffmpeg` may encounter metadata/chaining issues. If pure Go streaming fails, the system will fallback to orchestrating an `ezstream` or `ffmpeg` subprocess.