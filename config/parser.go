package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadSchedule(path string) (*ScheduleConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schedule config: %w", err)
	}

	var cfg ScheduleConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schedule config: %w", err)
	}

	return &cfg, nil
}

func LoadPersonas(path string) (*PersonasConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read personas config: %w", err)
	}

	var cfg PersonasConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal personas config: %w", err)
	}

	return &cfg, nil
}
