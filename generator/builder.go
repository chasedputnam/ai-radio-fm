package generator

import (
	"fmt"
	"strings"

	"github.com/chaseputnam/ai-radio-fm/config"
)

type PromptBuilder struct{}

func (b *PromptBuilder) BuildSystemPrompt(persona *config.Persona) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("You are %s. %s\n\nRules:\n", persona.Name, persona.Philosophy))
	for _, rule := range persona.PromptRules {
		builder.WriteString(fmt.Sprintf("- %s\n", rule))
	}
	return builder.String()
}

func (b *PromptBuilder) BuildUserPrompt(show *config.Show, ledgerHistory []string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("You are currently hosting '%s'. %s\n\n", show.Name, show.Description))
	
	if len(ledgerHistory) > 0 {
		builder.WriteString("Recent station history:\n")
		for _, entry := range ledgerHistory {
			builder.WriteString(fmt.Sprintf("- %s\n", entry))
		}
	}
	
	builder.WriteString("\nPlease write a short talk break script (no stage directions, just the spoken text).")
	return builder.String()
}
