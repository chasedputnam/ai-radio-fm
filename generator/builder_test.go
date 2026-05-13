package generator

import (
	"testing"
	"github.com/chaseputnam/ai-radio-fm/config"
)

func TestPromptBuilder(t *testing.T) {
	persona := &config.Persona{
		Name: "Nyx",
		Philosophy: "Dreams, night philosophy",
		PromptRules: []string{"Speak softly", "Be poetic"},
	}
	
	show := &config.Show{
		Name: "The Night Garden",
		Description: "Music for dreaming",
	}
	
	builder := &PromptBuilder{}
	sysPrompt := builder.BuildSystemPrompt(persona)
	
	if sysPrompt == "" {
		t.Error("Expected system prompt, got empty")
	}
	
	userPrompt := builder.BuildUserPrompt(show, []string{"Played track A", "Talked about moon"})
	if userPrompt == "" {
		t.Error("Expected user prompt, got empty")
	}
}
