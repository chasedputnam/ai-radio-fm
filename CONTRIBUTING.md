# Contributing to ai-radio-fm

Thanks for your interest in contributing! This guide covers everything you need to get started.

## Ways to Contribute

- **Report bugs** — open a [Bug Report](https://github.com/chaseputnam/ai-radio-fm/issues/new?template=bug_report.yml)
- **Request features** — open a [Feature Request](https://github.com/chaseputnam/ai-radio-fm/issues/new?template=feature_request.yml)
- **Submit pull requests** — bug fixes, new features, documentation improvements
- **Improve docs** — fix typos, clarify setup steps, add examples

## Development Setup

### Prerequisites

- Go 1.21+
- `espeak-ng` — required for TTS phonemization
- `onnxruntime` shared library — required for local TTS inference
- `ffmpeg` — optional, for MP3 encoding
- Icecast server — `brew install icecast` (macOS) or `apt-get install icecast2`
- Anthropic API key — set as `ANTHROPIC_API_KEY`
- [go-kokoro-tts](https://github.com/chasedputnam/go-kokoro-tts) — cloned alongside this repo
- [go-music-gen](https://github.com/kortexa-ai/go-music-gen) — optional, for music generation

### Getting Started

```bash
git clone https://github.com/chaseputnam/ai-radio-fm.git
cd ai-radio-fm

# Clone go-kokoro-tts alongside (required for TTS)
git clone https://github.com/chasedputnam/go-kokoro-tts.git ../go-kokoro-tts

# Download dependencies
go mod download

# Build the binary
go build -o airadio .

# Run tests
go test ./...
```

### Project Structure

```
ai-radio-fm/
├── cmd/                    # CLI commands (start, generate)
├── api/                    # HTTP API server (/now-playing, /schedule, /health)
├── operator/               # Background daemon — inventory and health loop
├── streamer/               # Ogg Vorbis streaming engine and Icecast client
├── generator/              # Anthropic, Kokoro TTS, and MusicGen clients
├── config/                 # Schedule and persona YAML loaders
├── ledger/                 # Append-only generation history log
├── stations/               # Per-station config directories
│   ├── example_station/    # Example station to copy from
│   └── .env.shared.example # Shared environment variable template
├── stations.sh             # Multi-station management script
├── install.sh              # One-step setup script
└── go.mod
```

### Build Commands

```bash
# Build for local platform
go build -o airadio .

# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run linter
go vet ./...

# Tidy dependencies
go mod tidy
```

## Making Changes

1. **Fork** the repository and create a branch from `main`:
   ```bash
   git checkout -b fix/describe-your-change
   ```

2. **Make your changes.** Keep commits focused — one logical change per commit.

3. **Run tests and lint** before pushing:
   ```bash
   go vet ./...
   go test -race ./...
   go mod tidy
   ```

4. **Open a pull request** against `main`. Fill out the PR template — describe what changed and how you tested it.

## Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Bug fix | `fix/short-description` | `fix/icecast-reconnect` |
| Feature | `feat/short-description` | `feat/multi-voice-personas` |
| Docs | `docs/short-description` | `docs/multi-station-setup` |
| Chore | `chore/short-description` | `chore/update-grpc` |

## Code Conventions

- Follow standard Go formatting — run `gofmt` before committing
- Keep package responsibilities focused: `streamer/` for audio I/O, `generator/` for content generation, `operator/` for the daemon loop — don't cross boundaries
- Add or update tests for any logic changes; test files live alongside source files (`*_test.go`)
- Avoid adding new dependencies unless necessary; discuss in an issue first
- Do not commit large binary files (`.ogg`, `.wav`, `.mp3`, `.flac`, `.onnx`, `.bin`)

## Commit Messages

Use the [Conventional Commits](https://www.conventionalcommits.org/) style:

```
feat: add per-show voice persona assignment
fix: handle icecast reconnect on dropped connection
docs: add multi-station setup walkthrough
chore: bump grpc to v1.72.0
test: add operator daemon inventory threshold test
```

## Security Issues

Do **not** open a public issue for security vulnerabilities. See [SECURITY.md](SECURITY.md) for the private reporting process.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
