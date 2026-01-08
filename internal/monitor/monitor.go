package monitor

import (
	"encoding/json"
	"os"
)

// Config represents the entire config file
type Config struct {
	Settings Settings `json:"settings"`
	Sites    []Site   `json:"sites"`
}

// Settings represents global app behavior
type Settings struct {
	CheckInterval int `json:"check_interval_seconds"`
	Timeout       int `json:"timeout_seconds"`
}

// Site represents an individual website to monitor
type Site struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string // This will be used later (Up/Down)
}

// LoadConfig reads and parses the config file
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}