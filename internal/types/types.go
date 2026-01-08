package types

import "time"

// Result holds the outcome of a status check
type Result struct {
	Name       string
	URL        string
	StatusCode int
	Latency    time.Duration
	IsUp       bool
}

// NotificationConfig holds settings for each notification channel
type NotificationConfig struct {
	Enabled  bool              `json:"enabled"`
	Channels []ChannelSettings `json:"channels"`
}

type ChannelSettings struct {
	Type     NotificationChannel `json:"type"`
	Enabled  bool                `json:"enabled"`
	Settings map[string]string   `json:"settings"`
}

// NotificationChannel represents different notification methods
type NotificationChannel string

const (
	ChannelDashboard NotificationChannel = "dashboard"
	ChannelEmail     NotificationChannel = "email"
	ChannelDiscord   NotificationChannel = "discord"
	ChannelTelegram  NotificationChannel = "telegram"
	ChannelSlack     NotificationChannel = "slack"
)
