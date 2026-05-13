package config

type Show struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	HostID      string `yaml:"host_id"`
	StartTime   string `yaml:"start_time"`
	EndTime     string `yaml:"end_time"`
	Description string `yaml:"description"`
}

type Persona struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	VoiceModel  string   `yaml:"voice_model"`
	Philosophy  string   `yaml:"philosophy"`
	PromptRules []string `yaml:"prompt_rules"`
}

type ScheduleConfig struct {
	Shows []Show `yaml:"shows"`
}

type PersonasConfig struct {
	Personas []Persona `yaml:"personas"`
}
