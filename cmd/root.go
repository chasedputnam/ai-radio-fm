package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "airadio",
	Short: "AI Radio FM is a 24/7 AI-powered music-forward internet radio station",
	Long:  `AI Radio FM runs autonomously, generating music and talk breaks, and streams continuously.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Root flags can be added here
}
