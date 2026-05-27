package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/chaseputnam/ai-radio-fm/api"
	"github.com/chaseputnam/ai-radio-fm/config"
	"github.com/chaseputnam/ai-radio-fm/generator"
	"github.com/chaseputnam/ai-radio-fm/ledger"
	"github.com/chaseputnam/ai-radio-fm/operator"
	"github.com/chaseputnam/ai-radio-fm/streamer"
	"github.com/spf13/cobra"
	"github.com/chasedputnam/go-kokoro-tts/pkg/model"
)

var stationName string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the AI Radio FM services",
	Long:  `Starts the core streaming stack, content daemons, and API server.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting AI Radio FM...")

		// --- Step 1: Load environment config ---
		cfg := config.LoadEnv()

		// --- Step 2: Resolve station paths and config files ---
		schedulePath := "config/schedule.yaml"
		personasPath := "config/personas.yaml"

		if stationName != "" {
			// Validate the station directory exists.
			stationDir := filepath.Join("stations", stationName)
			if _, err := os.Stat(stationDir); os.IsNotExist(err) {
				log.Fatalf("Station directory %q does not exist", stationDir)
			}

			paths := config.StationPathsFor(stationName)
			schedulePath = paths.ScheduleFile
			personasPath = paths.PersonasFile

			// Override paths with station-specific values unless env vars are set.
			if os.Getenv("CONTENT_DIR") == "" {
				cfg.ContentDir = paths.ContentDir
			}
			if os.Getenv("ARCHIVE_DIR") == "" {
				cfg.ArchiveDir = paths.ArchiveDir
			}
			if os.Getenv("LEDGER_PATH") == "" {
				cfg.LedgerPath = paths.LedgerPath
			}
			// Default mount to station name unless ICECAST_MOUNT is set.
			if os.Getenv("ICECAST_MOUNT") == "" {
				cfg.IcecastMount = stationName
			}

			log.Printf("Station: %q (dir: %s)", stationName, stationDir)
		}

		// --- Step 3: Load schedule and personas (fatal on error) ---
		schedule, err := config.LoadSchedule(schedulePath)
		if err != nil {
			log.Fatalf("Failed to load schedule: %v", err)
		}
		personas, err := config.LoadPersonas(personasPath)
		if err != nil {
			log.Fatalf("Failed to load personas: %v", err)
		}

		// --- Step 4: Ensure required directories exist ---
		for _, dir := range []string{cfg.ContentDir, cfg.ArchiveDir, filepath.Dir(cfg.LedgerPath)} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("Failed to create directory %q: %v", dir, err)
			}
		}

		// --- Step 5: Init ledger ---
		l := ledger.NewLedger(cfg.LedgerPath)

		// --- Step 6: Init playlist with now-playing callback ---
		// API server is needed for the callback; init it before the playlist.
		apiServer := api.NewServer(cfg.APIAddr, schedule)

		playlist := streamer.NewPlaylist()
		playlist.OnTrackChange = func(item streamer.PlaylistItem) {
			apiServer.UpdateNowPlaying(api.NowPlayingInfo{
				ShowID: item.ShowID,
				Track:  filepath.Base(item.FilePath),
			})
		}

		// --- Step 7: Init archiver ---
		archiver := streamer.NewArchiver(cfg.ArchiveDir, playlist)

		// --- Step 8: Init Icecast streamer client ---
		icecastCfg := streamer.IcecastConfig{
			Host:     cfg.IcecastHost,
			Port:     cfg.IcecastPort,
			Mount:    cfg.IcecastMount,
			User:     cfg.IcecastUser,
			Password: cfg.IcecastPassword,
		}
		streamClient := streamer.NewClient(icecastCfg)

		// --- Step 9: Init TTS renderer (gRPC sidecar or local ONNX) ---
		var tts generator.TTSRenderer
		if cfg.TTSGRPCAddr != "" {
			// Use the shared gRPC TTS sidecar.
			g, err := generator.NewGRPCTTSRenderer(cfg.TTSGRPCAddr)
			if err != nil {
				log.Printf("Warning: Failed to connect to TTS sidecar at %q: %v — TTS disabled", cfg.TTSGRPCAddr, err)
			} else {
				tts = g
				defer g.Close()
				log.Printf("TTS: using gRPC sidecar at %s", cfg.TTSGRPCAddr)
			}
		} else {
			// Fall back to local in-process ONNX engine.
			if _, err := os.Stat(cfg.KokoroLibPath); os.IsNotExist(err) {
				log.Printf("Warning: Kokoro lib not found at %q — TTS disabled", cfg.KokoroLibPath)
			} else if _, err := os.Stat(cfg.KokoroModelPath); os.IsNotExist(err) {
				log.Printf("Warning: Kokoro model not found at %q — TTS disabled", cfg.KokoroModelPath)
			} else {
				opts := model.EngineOptions{UseCoreML: cfg.KokoroCoreML}
				k, err := generator.NewKokoroTTS(cfg.KokoroLibPath, cfg.KokoroModelPath, "", cfg.KokoroVoiceDir, opts)
				if err != nil {
					log.Printf("Warning: Failed to init Kokoro TTS: %v — TTS disabled", err)
				} else {
					tts = k
					defer k.Close()
					log.Printf("TTS: using local ONNX engine")
				}
			}
		}

		// --- Step 10: Init generator clients ---
		anthropic := generator.NewAnthropicClientWithBaseURL(cfg.AnthropicAPIKey, cfg.AnthropicBaseURL)
		musicClient := generator.NewMusicGenClient(cfg.MusicGenURL, cfg.MusicGenFormat)

		// --- Step 11: Init ContentManager ---
		cm := generator.NewContentManager(anthropic, tts, musicClient, l, playlist, personas, cfg.ContentDir, cfg.MusicGenDuration)

		// --- Step 12: Init stream monitor and daemon ---
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sm := NewAppStreamMonitor(ctx, streamClient, icecastCfg, apiServer, playlist, archiver)

		daemon := operator.NewDaemon(cm, sm, schedule, cfg.InventoryThreshold)

		// --- Step 13: Start goroutines ---
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("API server error: %v", err)
			}
		}()

		// TeeReader feeds both Icecast and the archiver from the same playlist bytes.
		tee := io.TeeReader(playlist, archiver)
		sm.SetTee(tee)
		sm.StartStream()

		daemon.Start(ctx, 15*time.Minute)

		fmt.Println("AI Radio FM started. Press Ctrl+C to stop.")

		// --- Step 14: Block on signal, then clean up ---
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		fmt.Println("\nShutting down AI Radio FM...")
		daemon.Stop()
		cancel()
		// Give stream and daemon goroutines time to observe context cancellation.
		time.Sleep(500 * time.Millisecond)
		if err := archiver.Close(); err != nil {
			log.Printf("Archiver close error: %v", err)
		}
	},
}

func init() {
	startCmd.Flags().StringVar(&stationName, "station", "", "Station name — loads config from stations/<name>/")
	rootCmd.AddCommand(startCmd)
}
