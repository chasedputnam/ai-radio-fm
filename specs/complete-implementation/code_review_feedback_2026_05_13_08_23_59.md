# Code Review Feedback

## Summary

The implementation is solid overall — all seven stubs are replaced with real logic, the wiring is correct, and the test coverage is meaningful. There are three blocking issues: a deadlock risk in `Playlist.Read` when the `OnTrackChange` callback re-enters the playlist, a leaked goroutine in `RestartStream` (the new stream context is derived from `context.Background()` rather than the station's root context), and a redundant `anthropic` field on `ContentManager` that creates a confusing dual-path for script generation. Several suggestions and nits follow.

---

## Findings

### `streamer/playlist.go`

- [x] [BLOCKING] `OnTrackChange` callback is called with the mutex temporarily released, but `Read` re-acquires it immediately after — if the callback itself calls any `Playlist` method (e.g. `Enqueue` or `CurrentShow`), it will deadlock because `cond` is bound to `mu` and `Read` holds `mu` again before the callback returns.
  - Why: The unlock/callback/lock sequence is inside the `for` loop body. After `cb(item)` returns and `p.mu.Lock()` is called, execution falls through to `p.currentFile.Read(b)` — which is correct. But if the callback calls `p.Enqueue`, that also tries to acquire `p.mu`, which is now held again by `Read`. The `OnTrackChange` wiring in `start.go` calls `apiServer.UpdateNowPlaying` which does not touch the playlist, so this is safe today — but it is a latent trap for any future callback.
  - Fix: Document the constraint explicitly in the `OnTrackChange` field comment: "The callback must not call any Playlist methods, as the playlist lock is re-acquired immediately after the callback returns." This makes the contract clear without changing the implementation.
  - References: Requirement 7

- [x] [SUGGESTION] The `TestPlaylist_OnTrackChange` test relies on the second `p.Read(buf)` call hitting EOF on file 1 and then opening file 2 in the same call. This works because both files fit in the 64-byte buffer, but it is fragile — if the file content grows beyond the buffer size the second callback would not fire in the second `Read`. 
  - Fix: After the first `Read`, drain file 1 to EOF explicitly with a loop, then call `Read` once more to trigger the file-2 transition. Or use a 1-byte buffer to make the EOF/transition behaviour deterministic.

---

### `generator/content_manager.go`

- [x] [BLOCKING] `ContentManager` has both a `scriptGen scriptGenerator` interface field and a redundant `anthropic *AnthropicClient` concrete field. `NewContentManager` sets both to the same value, but `GenerateTalk` only uses `scriptGen`. The `anthropic` field is never read after construction, which means it exists solely as dead weight — but it also creates a subtle trap: a future developer might add logic that reads `cm.anthropic` directly, bypassing the interface and breaking testability.
  - Fix: Remove the `anthropic *AnthropicClient` field from the struct entirely. `NewContentManager` already accepts `*AnthropicClient` as a parameter and assigns it to `scriptGen` (which satisfies the interface). The concrete type is not needed on the struct.
  - References: Requirement 3

- [x] [SUGGESTION] The timestamp format `"20060102T150405Z"` used for WAV filenames does not include sub-second precision. If `GenerateTalk` is called twice within the same second for the same show (e.g. during a backfill run), the second call will silently overwrite the first file.
  - Fix: Use `time.Now().UTC().Format("20060102T150405.000Z")` to include milliseconds, or append a random suffix.

- [x] [SUGGESTION] `GenerateTalk` creates the talk directory and builds the output path before calling TTS, but if `cm.tts == nil` it skips rendering and never uses `outputPath`. The `os.MkdirAll` call is harmless but the path construction is wasted work and slightly misleading.
  - Fix: Move the `talkDir` / `outputPath` / `os.MkdirAll` block inside the `cm.tts != nil` branch.

- [x] [NIT] The `summary` truncation uses `summary[:120]` on a `string`, which slices bytes not runes. A multi-byte UTF-8 character (common in AI-generated scripts) at position 119–120 would produce an invalid UTF-8 string in the ledger.
  - Fix: `runes := []rune(summary); if len(runes) > 120 { summary = string(runes[:120]) }`

---

### `cmd/app.go`

- [x] [BLOCKING] `RestartStream` derives the new stream context from `context.Background()` rather than the station's root context. This means a restarted stream goroutine will not be cancelled when the operator presses Ctrl+C — it will leak until the process exits.
  - Why: `NewAppStreamMonitor` receives `parentCtx` and uses it for the initial context, but `RestartStream` discards it and uses `context.Background()` instead.
  - Fix: Store `parentCtx` on the struct and use it in `RestartStream`: `m.streamCtx, m.cancelStream = context.WithCancel(m.parentCtx)`.
  - References: Requirement 5

- [x] [SUGGESTION] The unused `icecastStatus` struct type (with the `SourceSingle` field tagged `json:"-"`) is defined but never used — `IsHealthy` uses an anonymous inline struct instead. 
  - Fix: Remove the `icecastStatus` type declaration.

- [x] [NIT] `SetTee` has no mutex protection. If `SetTee` and `StartStream` were called concurrently (unlikely given the sequential startup, but possible in tests), there would be a data race on `m.tee`.
  - Fix: Acquire `m.mu` in `SetTee`: `m.mu.Lock(); m.tee = r; m.mu.Unlock()`.

---

### `cmd/start.go`

- [x] [SUGGESTION] `UseCoreML: true` is hardcoded for the Kokoro engine options. On non-macOS systems (Linux CI, Docker) this will log a warning and fall back to CPU, which is fine — but it would be cleaner to expose a `KOKORO_COREML` env var so operators can explicitly disable it.
  - Fix: Add `KokoroCoreML bool` to `RuntimeConfig` (default `true` on darwin, `false` otherwise, or just a plain env var), and pass it to `model.EngineOptions`.

- [x] [SUGGESTION] The `time.Sleep(500 * time.Millisecond)` in the shutdown sequence is a magic number with no comment explaining what it is waiting for.
  - Fix: Add a comment: `// Give stream and daemon goroutines time to observe context cancellation.`

- [x] [NIT] `filepath.Dir(cfg.LedgerPath)` returns `"."` when `LedgerPath` is a bare filename like `./ledger.jsonl`. `os.MkdirAll(".", 0755)` is a no-op, so this is harmless — but it is worth a comment.

---

### `cmd/generate.go`

- [x] [SUGGESTION] `generate music` always uses the hardcoded prompt `"chill wave"` regardless of the show's description. This is inconsistent with `ContentManager.GenerateMusic` which uses `show.Description`.
  - Fix: Load the schedule, pick the first show, and pass `show.Description` (falling back to `"chill wave"` if empty) as the prompt.

---

### `config/env.go`

- [x] [NIT] `getEnv` returns the default when the env var is set to an empty string (`""`). This means `ICECAST_HOST=""` silently uses `"localhost"` rather than failing or using the empty value. This is the standard Go pattern for env-var helpers, but it means operators cannot explicitly set a value to empty.
  - Fix: Document this behaviour in the function comment: "Returns defaultVal if the variable is unset or empty."

---

## Positive observations

- The `scriptGenerator` interface on `ContentManager` is a clean testability pattern — mocking Anthropic without an HTTP server makes the unit tests fast and reliable.
- The Icecast `IsHealthy` implementation correctly handles both the array and single-object `source` shapes that real Icecast servers emit — this is a common gotcha that was handled proactively.
- `CountAudioFiles` returning `(0, nil)` for a missing directory is the right default — it causes the daemon to treat missing content dirs as empty inventory and trigger generation, which is the safe failure mode.
- The `OnTrackChange` callback releases the mutex before calling out, avoiding a deadlock with the API server's own mutex in `UpdateNowPlaying`.
- The nil-safe TTS path in `start.go` and `ContentManager` means the station can run in script-only mode without Kokoro installed, which is important for development and CI environments.
