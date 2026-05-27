# Changelog

All notable changes to ai-radio-fm are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [1.0.0] - 2026-05-28

### Added
- Single-binary Go architecture orchestrating audio streaming, content generation, and station management
- Native Ogg Vorbis streaming to Icecast server via `streamer/` package
- Stream archival — intercepts and records Ogg Vorbis data to rolling hourly files
- Anthropic Claude API integration for AI-generated talk script writing (`generator/`)
- Kokoro TTS integration for voice rendering — supports local ONNX engine and gRPC sidecar (`generator/`)
- [go-music-gen](https://github.com/kortexa-ai/go-music-gen) integration for AI music generation (`generator/`)
- Operator daemon — background health and inventory loop, ticks every 15 minutes (`operator/`)
- HTTP API server exposing `/now-playing`, `/schedule`, and `/health` endpoints (`api/`)
- Multi-station support — run up to five independent stations on one machine via `stations.sh`
- Per-station config directories under `stations/<name>/` with isolated content, archive, and ledger
- Shared gRPC TTS sidecar support — one `tts-server` process serves all stations
- `airadio start` — start a single station (legacy) or named station (`--station`)
- `airadio generate talk` — manually generate a talk segment
- `airadio generate music` — manually generate a music track with optional `--description` and `--duration`
- YAML-based schedule and persona configuration (`schedule.yaml`, `personas.yaml`)
- Environment variable configuration with sensible defaults for all runtime parameters
- `stations.sh` management script — `start`, `stop`, `status`, `restart`, `logs` commands
- Conflict detection for port and Icecast mount point collisions across stations
- `install.sh` — one-step setup script

### CLI Commands
| Command | Description |
|---------|-------------|
| `airadio start` | Start station using `config/` (single-station mode) |
| `airadio start --station <name>` | Start named station from `stations/<name>/` |
| `airadio generate talk` | Generate a talk segment manually |
| `airadio generate music` | Generate a music track manually |
| `airadio generate music --description <desc> --duration <secs>` | Generate music with explicit parameters |

---

[Unreleased]: https://github.com/chaseputnam/ai-radio-fm/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/chaseputnam/ai-radio-fm/releases/tag/v1.0.0
