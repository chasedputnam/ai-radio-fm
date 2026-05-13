# Tasks

- [x] 1. Add `RuntimeConfig` and `LoadEnv` to the config package
  - Create `config/env.go` with a `RuntimeConfig` struct containing all fields: `AnthropicAPIKey`, `IcecastHost`, `IcecastPort`, `IcecastMount`, `IcecastUser`, `IcecastPassword`, `ContentDir`, `KokoroLibPath`, `KokoroModelPath`, `KokoroVoiceDir`, `MusicGenURL`, `ArchiveDir`, `LedgerPath`, `APIAddr`
  - `LoadEnv()` reads each field from its environment variable with the following defaults: `IcecastHost=localhost`, `IcecastPort=8000`, `IcecastMount=stream`, `IcecastUser=source`, `IcecastPassword=hackme`, `ContentDir=./content`, `KokoroLibPath=/opt/homebrew/lib/libonnxruntime.dylib`, `KokoroModelPath=./go-kokoro-tts/kokoro-v0_19.onnx`, `KokoroVoiceDir=./go-kokoro-tts/voices`, `MusicGenURL=http://localhost:8002`, `ArchiveDir=./archive`, `LedgerPath=./ledger.jsonl`, `APIAddr=:8001`
  - References: Requirement 6
  - [x] 1.1 Write `config/env.go` with `RuntimeConfig` struct and `LoadEnv()` function
  - [x] 1.2 Write `config/env_test.go` with table-driven tests using `t.Setenv` covering defaults and overrides for each field

- [x] 2. Add content storage helpers to the generator package
  - Create `generator/storage.go` with `TalkDir(contentDir, showID string) string`, `MusicDir(contentDir, showID string) string`, and `CountAudioFiles(dir string) (int, error)` (counts `.wav` and `.ogg` files)
  - References: Requirement 2
  - [x] 2.1 Write `generator/storage.go` with the three functions
  - [x] 2.2 Write `generator/storage_test.go` using `t.TempDir()` with fixture files to verify directory paths and file counting, including the case where the directory does not exist (should return 0, nil)

- [x] 3. Add `OnTrackChange` callback to `streamer.Playlist`
  - Add `OnTrackChange func(item PlaylistItem)` field to the `Playlist` struct in `streamer/playlist.go`
  - In the `Read` method, when `p.currentFile` transitions from nil to a newly opened file, call `p.OnTrackChange(nextItem)` if the field is non-nil (call before releasing the lock or use a local copy to avoid holding the lock during the callback)
  - References: Requirement 7
  - [x] 3.1 Add the `OnTrackChange` field and the call site in `streamer/playlist.go`
  - [x] 3.2 Add a test in `streamer/playlist_test.go` that enqueues two temp files, reads through both, and asserts the callback fires once per file with the correct `PlaylistItem`

- [x] 4. Implement `generator.ContentManager`
  - Create `generator/content_manager.go` implementing `operator.ContentManager` with real `GenerateTalk`, `GenerateMusic`, and `InventoryLevel` logic as described in the design
  - The struct holds `*AnthropicClient`, `*KokoroTTS` (nullable), `*MusicGenClient`, `*ledger.Ledger`, `*streamer.Playlist`, `*config.PersonasConfig`, `contentDir string`, and `*PromptBuilder`
  - `GenerateTalk`: look up persona → read last 5 ledger entries → build prompts → call Anthropic → render TTS (skip if tts is nil) → write ledger entry → enqueue playlist item
  - `GenerateMusic`: request generation → wait for completion → download track → enqueue playlist item
  - `InventoryLevel`: sum `CountAudioFiles` for talk and music dirs; return 0 on error
  - References: Requirements 2, 3
  - [x] 4.1 Write `generator/content_manager.go` with the `ContentManager` struct and constructor `NewContentManager(...)`
  - [x] 4.2 Implement `GenerateTalk` including persona lookup, ledger read, prompt building, Anthropic call, conditional TTS render, ledger append, and playlist enqueue
  - [x] 4.3 Implement `GenerateMusic` including request, poll, download, and playlist enqueue
  - [x] 4.4 Implement `InventoryLevel` using the storage helpers
  - [x] 4.5 Write `generator/content_manager_test.go` with mock `AnthropicClient` and mock `KokoroTTS` (define a `TTSRenderer` interface in `content_manager.go` so the real `*KokoroTTS` and a mock both satisfy it); assert ledger entries are written and playlist items are enqueued for `GenerateTalk`; assert playlist item is enqueued for `GenerateMusic` using an `httptest.Server`; assert `InventoryLevel` returns correct counts from a temp directory

- [x] 5. Implement `AppStreamMonitor`
  - Replace the stub `AppStreamMonitor` in `cmd/app.go` with a real implementation
  - Add fields: `client *streamer.Client`, `icecastCfg streamer.IcecastConfig`, `apiServer *api.Server`, `cancelStream context.CancelFunc`, `playlist *streamer.Playlist`, `archiver *streamer.Archiver`, `mu sync.Mutex`
  - `IsHealthy`: GET `http://{host}:{port}/status-json.xsl`, parse the minimal Icecast JSON to check if the configured mount is listed as an active source; return false on any error
  - `RestartStream`: under mutex, cancel existing stream context, call `apiServer.UpdateHealth(false)`, start new goroutine with fresh context calling `io.TeeReader(playlist, archiver)` → `client.Stream`, call `apiServer.UpdateHealth(true)` after goroutine starts
  - References: Requirement 5
  - [x] 5.1 Define the full `AppStreamMonitor` struct with all fields and a constructor `NewAppStreamMonitor(...)`
  - [x] 5.2 Implement `IsHealthy` with the Icecast status endpoint check
  - [x] 5.3 Implement `RestartStream` with context cancellation and goroutine restart
  - [x] 5.4 Write tests for `IsHealthy` using `httptest.NewServer` returning mock Icecast JSON for both healthy and unhealthy cases

- [x] 6. Wire up `cmd/start.go`
  - Replace the hardcoded mock schedule and stub content manager with the full startup sequence from the design
  - Sequence: `LoadEnv` → `LoadSchedule` + `LoadPersonas` (exit on error) → init ledger → init playlist with `OnTrackChange` → init archiver → init streamer client → init API server → init KokoroTTS (nil-safe) → init ContentManager → init Daemon → `TeeReader` → start goroutines → block on signal → cleanup
  - References: Requirements 1, 4, 6
  - [x] 6.1 Add `config.LoadEnv()` call and replace all hardcoded Icecast/path values with `RuntimeConfig` fields
  - [x] 6.2 Add `config.LoadSchedule` and `config.LoadPersonas` calls with fatal error handling
  - [x] 6.3 Init `ledger.NewLedger`, `streamer.NewArchiver`, and wire `OnTrackChange` to call `apiServer.UpdateNowPlaying`
  - [x] 6.4 Init `generator.NewKokoroTTS` with nil-safe fallback (log warning if paths missing, pass nil tts to ContentManager)
  - [x] 6.5 Replace `AppContentManager{}` with `generator.NewContentManager(...)` and replace `AppStreamMonitor{}` with `NewAppStreamMonitor(...)`
  - [x] 6.6 Replace direct `playlist` pass to `streamClient.Stream` with `io.TeeReader(playlist, archiver)`
  - [x] 6.7 Add `archiver.Close()` call in the shutdown sequence after context cancel

- [x] 7. Fix `cmd/generate.go`
  - Replace hardcoded mock API key, persona, and show with real config loading
  - References: Requirement 1, 3
  - [x] 7.1 Add `config.LoadEnv()` call and use `cfg.AnthropicAPIKey` for `NewAnthropicClient`
  - [x] 7.2 Load schedule and personas from config files; use the first show and its associated persona; print a clear error and return if config files are missing

- [x] 8. Ensure content directories are created at startup
  - In `cmd/start.go`, after loading config, call `os.MkdirAll` for `cfg.ContentDir`, `cfg.ArchiveDir`, and the ledger file's parent directory
  - References: Requirement 2
  - [x] 8.1 Add `os.MkdirAll` calls for all required directories in the startup sequence, with fatal error handling

- [x] 9. Verify full build and test suite passes
  - Run `go build ./...` and `go test ./...` and fix any compilation errors or test failures introduced by the above changes
  - References: All requirements
  - [x] 9.1 Run `go build ./...` and resolve any import or type errors
  - [x] 9.2 Run `go test ./...` and confirm all existing and new tests pass
