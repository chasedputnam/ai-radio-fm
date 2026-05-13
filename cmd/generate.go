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

var generateMusicCmd = &cobra.Command{
	Use:   "music",
	Short: "Generate music tracks",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Generating music content...")

		cfg := config.LoadEnv()

		schedule, err := config.LoadSchedule("config/schedule.yaml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading schedule: %v\n", err)
			os.Exit(1)
		}

		prompt := "chill wave"
		if len(schedule.Shows) > 0 && schedule.Shows[0].Description != "" {
			prompt = schedule.Shows[0].Description
		}

		client := generator.NewMusicGenClient(cfg.MusicGenURL)

		_, err = client.RequestGeneration(context.Background(), prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Music generation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Music generation requested successfully.")
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(generateTalkCmd)
	generateCmd.AddCommand(generateMusicCmd)
}
