package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
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

// Result holds the outcome of a status check
type Result struct {
	Name       string
	URL        string
	StatusCode int
	Latency    time.Duration
	IsUp       bool
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

// CheckSite pings a single URL and returns a Result
func CheckSite(site Site, timeout int) Result {
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(site.URL)
	latency := time.Since(start)

	if err != nil || resp.StatusCode >= 400 {
		return Result{
			Name:       site.Name,
			URL:        site.URL,
			StatusCode: 0,
			IsUp:       false,
			Latency:    latency,
		}
	}
	defer resp.Body.Close()

	return Result{
		Name:       site.Name,
		URL:        site.URL,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		IsUp:       true,
	}
}
