package database

import "time"

// Site represents a monitored website
type Site struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Enabled       bool       `json:"enabled"`
	CurrentStatus string     `json:"current_status"` // "up" or "down"
	LastChecked   *time.Time `json:"last_checked"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AppSettings represents application-wide settings
type AppSettings struct {
	ID                   int       `json:"id"`
	CheckIntervalSeconds int       `json:"check_interval_seconds"`
	TimeoutSeconds       int       `json:"timeout_seconds"`
	RetentionDays        int       `json:"retention_days"`
	SLATargetPercentage  float64   `json:"sla_target_percentage"`
	EnableHistory        bool      `json:"enable_history"`
	Theme                string    `json:"theme"` // "dark" or "light"
	Timezone             string    `json:"timezone"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// UptimeRecord represents a single uptime check
type UptimeRecord struct {
	ID           int       `json:"id"`
	SiteID       int       `json:"site_id"`
	SiteName     string    `json:"site_name"`
	Timestamp    time.Time `json:"timestamp"`
	StatusCode   int       `json:"status_code"`
	IsUp         bool      `json:"is_up"`
	LatencyMs    int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// UptimeSummary represents daily uptime statistics
type UptimeSummary struct {
	ID               int       `json:"id"`
	SiteID           int       `json:"site_id"`
	SiteName         string    `json:"site_name"`
	Date             time.Time `json:"date"`
	TotalChecks      int       `json:"total_checks"`
	SuccessfulChecks int       `json:"successful_checks"`
	FailedChecks     int       `json:"failed_checks"`
	UptimePercentage float64   `json:"uptime_percentage"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	MinLatencyMs     int64     `json:"min_latency_ms"`
	MaxLatencyMs     int64     `json:"max_latency_ms"`
	DowntimeMinutes  int64     `json:"downtime_minutes"`
	IncidentCount    int       `json:"incident_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// DowntimeIncident represents a downtime period
type DowntimeIncident struct {
	ID              int        `json:"id"`
	SiteID          int        `json:"site_id"`
	SiteName        string     `json:"site_name"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	DurationSeconds int64      `json:"duration_seconds"`
	IncidentCount   int        `json:"incident_count"`
	RootCause       string     `json:"root_cause,omitempty"`
}

// NotificationLog represents a sent notification
type NotificationLog struct {
	ID             int       `json:"id"`
	NotificationID string    `json:"notification_id"`
	SiteID         int       `json:"site_id"`
	Type           string    `json:"type"` // "email", "discord", etc
	Message        string    `json:"message"`
	Severity       string    `json:"severity"` // "error", "warning", "info", "success"
	SentAt         time.Time `json:"sent_at"`
	Status         string    `json:"status"` // "sent", "failed", "pending"
	RetryCount     int       `json:"retry_count"`
}

// AuditLog represents user actions and changes
type AuditLog struct {
	ID         int       `json:"id"`
	Action     string    `json:"action"`      // "incident_started", "incident_closed", etc
	EntityType string    `json:"entity_type"` // "incident", "site", "setting"
	EntityID   int       `json:"entity_id"`
	OldValue   string    `json:"old_value,omitempty"` // JSON
	NewValue   string    `json:"new_value,omitempty"` // JSON
	UserID     string    `json:"user_id"`
	Timestamp  time.Time `json:"timestamp"`
}
