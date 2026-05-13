# Tasks: Kokoro Go Integration

## 1. Project Configuration
- [x] 1.1 Update `go.mod` to include the local `go-kokoro-tts` library
  - Use `go mod edit -replace` to point to the local directory
  - Run `go mod tidy` to resolve dependencies
  - References: Requirement 1

## 2. TTS Engine Refactor
- [x] 2.1 Update `generator/tts.go` struct definition
  - Add `api.Pipeline`, `model.ONNXEngine`, and `voice.VoiceLoader` fields
  - Remove Python-specific fields
  - References: Requirement 1, 2
- [x] 2.2 Refactor `NewKokoroTTS` constructor
  - Initialize the ONNX engine and API pipeline
  - Load the vocabulary file
  - References: Requirement 1, 3
- [x] 2.3 Refactor `Render` method
  - Implement synthesis using `pipeline.Synthesize`
  - Implement WAV file writing using the `audio` package from the library
  - References: Requirement 2

## 3. Testing and Validation
- [x] 3.1 Create `generator/tts_integration_test.go`
  - Add a test case to verify end-to-end synthesis to a file
  - Use real model assets from the `go-kokoro-tts` directory for the test
  - References: Requirement 2
- [x] 3.2 Verify project builds and tests pass
  - Run `go build ./...`
  - Run `go test ./generator/...`
  - References: Requirement 1
