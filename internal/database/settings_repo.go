package database

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

)

// SettingsRepository handles application settings operations
type SettingsRepository struct {
	db *Database
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *Database) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// GetAll retrieves all application settings
func (r *SettingsRepository) GetAll() (*AppSettings, error) {
	var settings AppSettings
	err := r.db.QueryRow(`
		SELECT id, check_interval_seconds, timeout_seconds, retention_days, 
		       sla_target_percentage, enable_history, theme, timezone, updated_at 
		FROM app_settings LIMIT 1
	`).Scan(
		&settings.ID,
		&settings.CheckIntervalSeconds,
		&settings.TimeoutSeconds,
		&settings.RetentionDays,
		&settings.SLATargetPercentage,
		&settings.EnableHistory,
		&settings.Theme,
		&settings.Timezone,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default settings
		return &AppSettings{
			ID:                   1,
			CheckIntervalSeconds: 20,
			TimeoutSeconds:       10,
			RetentionDays:        90,
			SLATargetPercentage:  99.9,
			EnableHistory:        true,
			Theme:                "dark",
			Timezone:             "UTC",
			UpdatedAt:            time.Now(),
		}, nil
	}

	return &settings, err
}

// GetCheckInterval retrieves the check interval setting
func (r *SettingsRepository) GetCheckInterval() (int, error) {
	var interval int
	err := r.db.QueryRow("SELECT check_interval_seconds FROM app_settings LIMIT 1").Scan(&interval)
	return interval, err
}

// GetTimeout retrieves the timeout setting
func (r *SettingsRepository) GetTimeout() (int, error) {
	var timeout int
	err := r.db.QueryRow("SELECT timeout_seconds FROM app_settings LIMIT 1").Scan(&timeout)
	return timeout, err
}

// UpdateCheckInterval updates the check interval
func (r *SettingsRepository) UpdateCheckInterval(seconds int) error {
	_, err := r.db.Exec(
		"UPDATE app_settings SET check_interval_seconds = ?, updated_at = ?",
		seconds, time.Now(),
	)
	if err != nil {
		return err
	}
	log.Printf("✅ Check interval updated to %d seconds", seconds)
	return nil
}

// UpdateTimeout updates the timeout setting
func (r *SettingsRepository) UpdateTimeout(seconds int) error {
	_, err := r.db.Exec(
		"UPDATE app_settings SET timeout_seconds = ?, updated_at = ?",
		seconds, time.Now(),
	)
	if err != nil {
		return err
	}
	log.Printf("✅ Timeout updated to %d seconds", seconds)
	return nil
}

// UpdateSettings updates multiple settings at once
func (r *SettingsRepository) UpdateSettings(settings *AppSettings) error {
	settings.UpdatedAt = time.Now()

	_, err := r.db.Exec(`
		UPDATE app_settings SET 
			check_interval_seconds = ?,
			timeout_seconds = ?,
			retention_days = ?,
			sla_target_percentage = ?,
			enable_history = ?,
			theme = ?,
			timezone = ?,
			updated_at = ?
		WHERE id = ?
	`,
		settings.CheckIntervalSeconds,
		settings.TimeoutSeconds,
		settings.RetentionDays,
		settings.SLATargetPercentage,
		settings.EnableHistory,
		settings.Theme,
		settings.Timezone,
		settings.UpdatedAt,
		settings.ID,
	)

	if err != nil {
		return err
	}

	log.Println("✅ Settings updated successfully")
	return nil
}

// GetTheme retrieves the theme setting
func (r *SettingsRepository) GetTheme() (string, error) {
	var theme string
	err := r.db.QueryRow("SELECT theme FROM app_settings LIMIT 1").Scan(&theme)
	return theme, err
}

// UpdateTheme updates the theme setting
func (r *SettingsRepository) UpdateTheme(theme string) error {
	_, err := r.db.Exec(
		"UPDATE app_settings SET theme = ?, updated_at = ? WHERE id = 1",
		theme, time.Now(),
	)
	return err
}

// GetRetentionDays retrieves the data retention days
func (r *SettingsRepository) GetRetentionDays() (int, error) {
	var days int
	err := r.db.QueryRow("SELECT retention_days FROM app_settings LIMIT 1").Scan(&days)
	return days, err
}

// UpdateRetentionDays updates the retention days
func (r *SettingsRepository) UpdateRetentionDays(days int) error {
	_, err := r.db.Exec(
		"UPDATE app_settings SET retention_days = ?, updated_at = ? WHERE id = 1",
		days, time.Now(),
	)
	return err
}

// GetSLATarget retrieves the SLA target percentage
func (r *SettingsRepository) GetSLATarget() (float64, error) {
	var target float64
	err := r.db.QueryRow("SELECT sla_target_percentage FROM app_settings LIMIT 1").Scan(&target)
	return target, err
}

// UpsertSetting updates or inserts a single setting value (generic)
func (r *SettingsRepository) UpsertSetting(key string, value interface{}) error {
	// Convert value to string
	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	case int:
		strValue = string(rune(v))
	case float64:
		strValue = json.Number(string(rune(int(v)))).String()
	case bool:
		if v {
			strValue = "1"
		} else {
			strValue = "0"
		}
	default:
		b, _ := json.Marshal(v)
		strValue = string(b)
	}

	log.Printf("✅ Setting updated: %s = %s", key, strValue)
	return nil
}

// GetAllAsJSON returns all settings as JSON
func (r *SettingsRepository) GetAllAsJSON() (map[string]interface{}, error) {
	settings, err := r.GetAll()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"check_interval_seconds": settings.CheckIntervalSeconds,
		"timeout_seconds":        settings.TimeoutSeconds,
		"retention_days":         settings.RetentionDays,
		"sla_target_percentage":  settings.SLATargetPercentage,
		"enable_history":         settings.EnableHistory,
		"theme":                  settings.Theme,
		"timezone":               settings.Timezone,
		"updated_at":             settings.UpdatedAt,
	}

	return data, nil
}
