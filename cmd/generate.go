package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/chaseputnam/ai-radio-fm/config"
	"github.com/chaseputnam/ai-radio-fm/generator"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Manually execute content generation workflows",
	Long:  `Generate talk scripts, audio, or music manually.`,
}

var generateTalkCmd = &cobra.Command{
	Use:   "talk",
	Short: "Generate talk scripts and audio",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating talk content...")

		cfg := config.LoadEnv()

		schedule, err := config.LoadSchedule("config/schedule.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading schedule: %v\n", err)
			os.Exit(1)
		}
		personas, err := config.LoadPersonas("config/personas.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading personas: %v\n", err)
			os.Exit(1)
		}

		if len(schedule.Shows) == 0 {
			fmt.Fprintln(os.Stderr, "No shows defined in schedule.yaml")
			os.Exit(1)
		}
		show := &schedule.Shows[0]

		// Find the persona for this show.
		var persona *config.Persona
		for i := range personas.Personas {
			if personas.Personas[i].ID == show.HostID {
				persona = &personas.Personas[i]
				break
			}
		}
		if persona == nil {
			fmt.Fprintf(os.Stderr, "No persona found for host_id %q\n", show.HostID)
			os.Exit(1)
		}

		if cfg.AnthropicAPIKey == "" {
			fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY is not set")
			os.Exit(1)
		}

		client := generator.NewAnthropicClientWithBaseURL(cfg.AnthropicAPIKey, cfg.AnthropicBaseURL)
		builder := &generator.PromptBuilder{}

		sys := builder.BuildSystemPrompt(persona)
		user := builder.BuildUserPrompt(show, []string{})

		script, err := client.GenerateScript(context.Background(), sys, user)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Script generation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated script for show %q (host: %s):\n\n%s\n", show.Name, persona.Name, script)
	},
}

var (
	musicDuration    float64
	musicDescription string
)

var generateMusicCmd = &cobra.Command{
	Use:   "music",
	Short: "Generate music tracks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating music content...")

		cfg := config.LoadEnv()

		// Resolve description: flag > schedule description > default.
		prompt := musicDescription
		if prompt == "" {
			schedule, err := config.LoadSchedule("config/schedule.yaml")
			if err == nil && len(schedule.Shows) > 0 && schedule.Shows[0].Description != "" {
				prompt = schedule.Shows[0].Description
			}
		}
		if prompt == "" {
			prompt = "ambient electronic music"
		}

		// Resolve duration: flag > env default.
		duration := musicDuration
		if duration <= 0 {
			duration = cfg.MusicGenDuration
		}

		client := generator.NewMusicGenClient(cfg.MusicGenURL, cfg.MusicGenFormat)

		// Use a temp dir to hold the generated file.
		tmpDir, err := os.MkdirTemp("", "airadio-music-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create temp dir: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generating: %q (%.0fs)\n", prompt, duration)

		filePath, err := client.Generate(context.Background(), prompt, tmpDir, duration)
		if err != nil {
			os.RemoveAll(tmpDir)
			fmt.Fprintf(os.Stderr, "Music generation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Music generated successfully: %s\n", filePath)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateTalkCmd)
	generateCmd.AddCommand(generateMusicCmd)

	generateMusicCmd.Flags().Float64VarP(&musicDuration, "duration", "d", 0,
		"Track duration in seconds (default: MUSICGEN_DURATION env var, fallback 90)")
	generateMusicCmd.Flags().StringVarP(&musicDescription, "description", "D", "",
		"Music style/genre description (default: first show's description from schedule.yaml)")
}
