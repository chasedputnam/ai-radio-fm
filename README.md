# AI Radio FM

A 24/7 AI-powered music-forward internet radio station written in Go. AI generates the music, writes short hosted breaks based on host personas and schedules, TTS speaks them, and the stream runs continuously.

## What is this?

AI Radio FM is the Golang-based successor to `writ-fm`. It provides a seamless, single-binary architecture that orchestrates:
- **Audio Streaming:** Native Ogg Vorbis streaming to an Icecast server with seamless playlist queueing and stream archival capabilities.
- **Content Generation:** Integrates with Anthropic's Claude API for script writing, Kokoro TTS for voice rendering, and `music-gen.server` for AI music generation.
- **Autonomous Operation:** An internal background loop ensures content buffers are always stocked and system health is maintained.
- **API Access:** Exposes real-time station status and scheduling data.
- **Multi-Station:** Run up to five independent stations on one machine, each with its own schedule, personas, and Icecast mount point, managed by a single orchestration script.

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│  airadio (Go Binary)                                         │
├──────────────────────────────────────────────────────────────┤
│  ► CLI layer (start [--station], generate)                   │
│  ► API Server (:8001) ─► /now-playing, /schedule, /health    │
│  ► Operator Daemon    ─► Background health & inventory loop  │
│  ► Streaming Engine   ─► Audio orchestration & Icecast client│
│  ► Stream Archiver    ─► Intercepts and records stream       │
│  ► Generator Clients  ─► Anthropic, Kokoro TTS, MusicGen     │
└──────────────────────────────────────────────────────────────┘
                               │
       ┌───────────────────────┴────────────────────────┐
       ▼                       ▼                        ▼
  Icecast Server         Local Audio Files          External APIs
```

### Multi-Station Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│  Machine                                                        │
│                                                                 │
│  Shared Services                                                │
│  ├── Icecast :8000  (/station_a, /station_b, /station_c, ...)  │
│  ├── MusicGen Server :8002                                      │
│  └── tts-server (gRPC) :50051                                   │
│                                                                 │
│  Station Processes (one per airadio binary)                     │
│  ├── airadio --station station_a  (API :8001)                   │
│  ├── airadio --station station_b  (API :8002)                   │
│  └── airadio --station station_c  (API :8003)                   │
└─────────────────────────────────────────────────────────────────┘
```

Each station is a fully independent OS process with its own config, content, archive, and ledger under `stations/<name>/`. The TTS sidecar loads the ONNX model once and serves all stations over gRPC, avoiding the memory cost of five separate inference engines.

---

## Prerequisites

- **Go 1.21+**
- **Icecast**: `brew install icecast` (macOS) or `apt-get install icecast2`
- **espeak-ng**: Required for TTS. `brew install espeak-ng` or `apt-get install espeak-ng`
- **ONNX Runtime**: Required for TTS. `brew install onnxruntime` or download from [onnxruntime releases](https://github.com/microsoft/onnxruntime/releases)
- **Anthropic API key**: Set as `ANTHROPIC_API_KEY`
- **go-kokoro-tts**: Clone alongside this repo — `git clone https://github.com/user/go-kokoro-tts.git ../go-kokoro-tts`

---

## Single Station Setup

### 1. Build

```bash
git clone https://github.com/chaseputnam/ai-radio-fm.git
cd ai-radio-fm
go build -o airadio .
```

### 2. Configure

```bash
mkdir -p config
cp config/schedule.yaml.example config/schedule.yaml
cp config/personas.yaml.example config/personas.yaml
```

Edit both files to define your shows and host personas. The `host_id` in each show must match a persona `id`.

### 3. Set environment variables

The station reads all configuration from environment variables. At minimum:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export ICECAST_PASSWORD=your_icecast_password
```

Full list of variables with defaults:

| Variable | Default | Description |
|---|---|---|
| `ANTHROPIC_API_KEY` | _(required)_ | Anthropic API key for script generation |
| `ANTHROPIC_BASE_URL` | `https://api.anthropic.com` | Anthropic API base URL. Override for alternative providers (LiteLLM, OpenRouter, etc.) |
| `ICECAST_HOST` | `localhost` | Icecast server hostname |
| `ICECAST_PORT` | `8000` | Icecast server port |
| `ICECAST_MOUNT` | `stream` | Icecast mount point |
| `ICECAST_USER` | `source` | Icecast source username |
| `ICECAST_PASSWORD` | `hackme` | Icecast source password |
| `MUSICGEN_URL` | `http://localhost:8002` | MusicGen server URL |
| `TTS_GRPC_ADDR` | _(empty)_ | gRPC TTS sidecar address. If set, uses sidecar instead of local ONNX |
| `KOKORO_LIB_PATH` | `/opt/homebrew/lib/libonnxruntime.dylib` | ONNX runtime shared library |
| `KOKORO_MODEL_PATH` | `./go-kokoro-tts/kokoro-v0_19.onnx` | Kokoro ONNX model |
| `KOKORO_VOICE_DIR` | `./go-kokoro-tts/voices` | Voice profile directory |
| `KOKORO_COREML` | `true` | Enable CoreML acceleration (macOS) |
| `INVENTORY_THRESHOLD` | `5` | Number of audio files to keep buffered per show before the daemon stops generating |
| `CONTENT_DIR` | `./content` | Generated audio output directory |
| `ARCHIVE_DIR` | `./archive` | Stream archive directory |
| `LEDGER_PATH` | `./ledger.jsonl` | Generation history log |
| `API_ADDR` | `:8001` | API server listen address |

### 4. Start

```bash
./airadio start
```

This starts the streaming engine (connecting to Icecast), the API server, and the operator daemon. The daemon immediately checks content inventory and begins generating talk segments and music if needed.

---

## First Start Behaviour

On a fresh station with no pre-generated content, there will be a brief silence on the Icecast mount while the first content is produced. Here is exactly what happens:

1. The station process starts and connects to Icecast. The playlist is empty.
2. The operator daemon ticks immediately (no waiting for the 15-minute interval).
3. It checks inventory — zero files found.
4. It calls the Anthropic API to generate a talk script, then renders it to a WAV file via the TTS sidecar (or local ONNX engine). This typically takes 30–90 seconds depending on script length and hardware.
5. It calls the MusicGen server to request a music track, polls until complete, and downloads the result. This typically takes 1–3 minutes depending on the MusicGen server queue.
6. Both files are enqueued on the playlist. Audio begins streaming to Icecast as soon as the first file is ready.
7. The daemon continues ticking every 15 minutes, generating more content whenever inventory drops below `INVENTORY_THRESHOLD` (default: 5).

**To avoid the initial silence**, you can drop any `.ogg` or `.wav` audio files directly into `content/music/<show_id>/` or `content/talk/<show_id>/` before starting. The daemon will count them as inventory and enqueue them immediately.

**If MusicGen is not running**, music generation will fail silently and the daemon will log the error. The station will still operate with talk segments only until MusicGen becomes available.

**Tuning the buffer**: set `INVENTORY_THRESHOLD` higher (e.g. `10`) to keep more content pre-generated and reduce the chance of the playlist running dry during a slow generation cycle.

---

## Multi-Station Setup

Up to five stations can run on one machine. Each station is an independent process with its own config directory. A shared Icecast instance handles all streams on separate mount points. A single gRPC TTS sidecar serves all stations.

### 1. Build both binaries

```bash
# Station binary
go build -o airadio .

# TTS sidecar (in the go-kokoro-tts repo)
go build -o ../go-kokoro-tts/tts-server ../go-kokoro-tts/cmd/tts-server/
```

### 2. Create station directories

Each station lives under `stations/<name>/`. Start from the example:

```bash
cp -r stations/example_station stations/my_station
```

Edit `stations/my_station/schedule.yaml` and `stations/my_station/personas.yaml` for your station's content.

### 3. Configure shared environment

```bash
cp stations/.env.shared.example stations/.env.shared
```

Edit `stations/.env.shared` with values shared across all stations:

```bash
ANTHROPIC_API_KEY=sk-ant-...
ICECAST_HOST=localhost
ICECAST_PORT=8000
ICECAST_USER=source
ICECAST_PASSWORD=your_password
TTS_GRPC_ADDR=localhost:50051
MUSICGEN_URL=http://localhost:8002
KOKORO_LIB_PATH=/opt/homebrew/lib/libonnxruntime.dylib
KOKORO_MODEL_PATH=./go-kokoro-tts/kokoro-v0_19.onnx
KOKORO_VOICE_DIR=./go-kokoro-tts/voices
```

### 4. Configure per-station environment

Each station needs a unique API port. The Icecast mount defaults to `/<station_name>` automatically.

```bash
cp stations/example_station/.env.example stations/my_station/.env
```

Edit `stations/my_station/.env`:

```bash
API_ADDR=:8001
# ICECAST_MOUNT defaults to /my_station — override here if needed
```

Give each station a different `API_ADDR` (`:8001`, `:8002`, `:8003`, etc.).

### 5. Start all stations

```bash
./stations.sh start
```

This will:
1. Detect all stations under `stations/` (any directory with a `schedule.yaml`)
2. Check for port and mount point conflicts — exits with an error if any are found
3. Start the `tts-server` gRPC sidecar
4. Start each station as a background process

### Station management commands

```bash
# Check what's running
./stations.sh status

# Stop everything
./stations.sh stop

# Restart a single station without touching others
./stations.sh restart my_station

# Tail a station's log
./stations.sh logs my_station
```

### Station directory layout

```
stations/
  .env.shared              # Shared env vars (Icecast, MusicGen, Anthropic, TTS addr)
  tts-server.pid           # PID of the running tts-server
  tts-server.log           # TTS sidecar output
  <name>/
    schedule.yaml          # Show definitions
    personas.yaml          # Host persona definitions
    .env                   # Per-station overrides (API_ADDR, ICECAST_MOUNT, etc.)
    content/
      talk/<show_id>/      # Generated WAV talk segments
      music/<show_id>/     # Downloaded music tracks
    archive/               # Rolling Ogg Vorbis archive files
    ledger.jsonl           # Append-only generation log
    station.pid            # PID of the running airadio process
    station.log            # Station stdout/stderr
```

---

## CLI Commands

```bash
# Start a single station (legacy mode, uses config/)
./airadio start

# Start a named station (uses stations/<name>/)
./airadio start --station my_station

# Generate a talk segment manually
./airadio generate talk

# Request a music track manually
./airadio generate music
```

---

## API Endpoints

Each station exposes its own HTTP API on its configured `API_ADDR`:

| Endpoint | Description |
|---|---|
| `GET /now-playing` | Currently playing track, show ID, and host name |
| `GET /schedule` | Full loaded schedule |
| `GET /health` | `{"status":"ok"}` when healthy |

---

## Stream Archival

The `StreamArchiver` intercepts the Ogg Vorbis data being sent to Icecast and writes it to disk simultaneously. Files roll over every hour or at show boundaries, named with a timestamp and show ID.

---

## Development and Testing

```bash
go test ./...
go fmt ./...
go mod tidy
```
