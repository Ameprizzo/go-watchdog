package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
	"sync"
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

// StatusStore keeps the latest results safe for concurrent access
type StatusStore struct {
	mu      sync.RWMutex
	Results []Result
}

// Update replaces the old results with new ones
func (s *StatusStore) Update(newResults []Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Results = newResults
}

// Get returns a copy of the current results
func (s *StatusStore) Get() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Results
}

var Store = &StatusStore{}

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

func (c *Config) AddSite(newSite Site) error {
    // In a real app, you'd add a Mutex here too
    c.Sites = append(c.Sites, newSite)
    
    // Persist to disk
    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile("config.json", data, 0644)
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
