# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | ✅        |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Use GitHub's private vulnerability reporting:

1. Go to the [Security tab](https://github.com/chaseputnam/ai-radio-fm/security) of this repository
2. Click **"Report a vulnerability"**
3. Fill in the details and submit

You can expect:
- **Acknowledgement** within 48 hours
- **Status update** within 7 days
- **Resolution or mitigation** as quickly as possible depending on severity

## What to Include

- Type of vulnerability (e.g. command injection, path traversal, unsafe deserialization)
- Full paths of affected source files
- Steps to reproduce
- Proof-of-concept or exploit code (if available)
- Impact assessment

## Scope

This project invokes external processes and handles user-supplied configuration. Relevant attack surfaces include:

- Environment variable and YAML config parsing (`config/`)
- Subprocess invocation via `espeak-ng` and `ffmpeg` (via go-kokoro-tts)
- gRPC client communication with the TTS sidecar (`generator/`)
- HTTP API server input handling (`api/`)
- File path handling for content, archive, and ledger directories
- Icecast source client connection handling (`streamer/`)

## Preferred Languages

English.
