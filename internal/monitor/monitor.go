package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
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

// StatusStore keeps the latest results safe for concurrent access
type StatusStore struct {
	mu            sync.RWMutex
	Results       []Result
	PreviousState map[string]bool // map[URL]IsUp
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

func NewStore() *StatusStore {
	return &StatusStore{
		PreviousState: make(map[string]bool),
	}
}

// CheckAndUpdateState checks if a site's status changed and updates the internal state
// Returns true if the status changed (for alerting), false otherwise
func (s *StatusStore) CheckAndUpdateState(url string, isUp bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get previous state (default to true if not found, so we don't alert on first check)
	previousIsUp, exists := s.PreviousState[url]

	// If it's the first time seeing this URL, just store the state and don't alert
	if !exists {
		s.PreviousState[url] = isUp
		return false
	}

	// Check if state changed
	stateChanged := previousIsUp != isUp

	// Update the state
	s.PreviousState[url] = isUp

	return stateChanged
}

var Store = NewStore()

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

// UpdateSettings changes global interval/timeout and saves to disk
func (c *Config) UpdateSettings(interval, timeout int) error {
	c.Settings.CheckInterval = interval
	c.Settings.Timeout = timeout
	return c.save()
}

// DeleteSite removes a site by its name and saves
func (c *Config) DeleteSite(name string) error {
	for i, s := range c.Sites {
		if s.Name == name {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			break
		}
	}
	return c.save()
}

// Private helper to save to config.json
func (c *Config) save() error {
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
