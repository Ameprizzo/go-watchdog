package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/types"
)

// Config represents the entire config file
type Config struct {
	Settings      Settings                  `json:"settings"`
	Sites         []Site                    `json:"sites"`
	Notifications *types.NotificationConfig `json:"notifications"`
	mu            sync.RWMutex
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

	// Initialize notifications if not present
	if config.Notifications == nil {
		config.Notifications = &types.NotificationConfig{
			Enabled:  true,
			Channels: []types.ChannelSettings{},
		}
	}

	return &config, nil
}

func (c *Config) AddSite(newSite Site) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Sites = append(c.Sites, newSite)
	return c.save()
}

// UpdateSettings changes global interval/timeout and saves to disk
func (c *Config) UpdateSettings(interval, timeout int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Settings.CheckInterval = interval
	c.Settings.Timeout = timeout
	return c.save()
}

// DeleteSite removes a site by its name and saves
func (c *Config) DeleteSite(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, s := range c.Sites {
		if s.Name == name {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			break
		}
	}
	return c.save()
}

// UpdateNotificationSettings updates notification configuration
func (c *Config) UpdateNotificationSettings(notifConfig *types.NotificationConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Notifications = notifConfig
	return c.save()
}

// AddNotificationChannel adds or updates a notification channel
func (c *Config) AddNotificationChannel(channel types.ChannelSettings) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if channel already exists
	found := false
	for i, ch := range c.Notifications.Channels {
		if ch.Type == channel.Type {
			c.Notifications.Channels[i] = channel
			found = true
			break
		}
	}

	if !found {
		c.Notifications.Channels = append(c.Notifications.Channels, channel)
	}

	return c.save()
}

// RemoveNotificationChannel removes a notification channel
func (c *Config) RemoveNotificationChannel(channelType types.NotificationChannel) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, ch := range c.Notifications.Channels {
		if ch.Type == channelType {
			c.Notifications.Channels = append(c.Notifications.Channels[:i], c.Notifications.Channels[i+1:]...)
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
func CheckSite(site Site, timeout int) types.Result {
	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(site.URL)
	latency := time.Since(start)

	if err != nil || resp.StatusCode >= 400 {
		return types.Result{
			Name:       site.Name,
			URL:        site.URL,
			StatusCode: 0,
			IsUp:       false,
			Latency:    latency,
		}
	}
	defer resp.Body.Close()

	return types.Result{
		Name:       site.Name,
		URL:        site.URL,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		IsUp:       true,
	}
}
