package database

import (
	"log"
	"time"
)

// UptimeRepository handles uptime record operations
type UptimeRepository struct {
	db *Database
}

// NewUptimeRepository creates a new uptime repository
func NewUptimeRepository(db *Database) *UptimeRepository {
	return &UptimeRepository{db: db}
}

// Create inserts a new uptime record
func (r *UptimeRepository) Create(record *UptimeRecord) error {
	result, err := r.db.Exec(`
		INSERT INTO uptime_records (site_id, timestamp, status_code, is_up, latency_ms, error_message)
		VALUES (?, ?, ?, ?, ?, ?)
	`, record.SiteID, record.Timestamp, record.StatusCode, record.IsUp, record.LatencyMs, record.ErrorMessage)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	record.ID = int(id)
	return nil
}

// GetBySiteID retrieves all uptime records for a site
func (r *UptimeRepository) GetBySiteID(siteID int, limit int) ([]UptimeRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, timestamp, status_code, is_up, latency_ms, error_message
		FROM uptime_records
		WHERE site_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, siteID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []UptimeRecord
	for rows.Next() {
		var record UptimeRecord
		if err := rows.Scan(&record.ID, &record.SiteID, &record.Timestamp, &record.StatusCode,
			&record.IsUp, &record.LatencyMs, &record.ErrorMessage); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// GetByDateRange retrieves uptime records within a date range
func (r *UptimeRepository) GetByDateRange(siteID int, startTime, endTime time.Time) ([]UptimeRecord, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, timestamp, status_code, is_up, latency_ms, error_message
		FROM uptime_records
		WHERE site_id = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`, siteID, startTime, endTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []UptimeRecord
	for rows.Next() {
		var record UptimeRecord
		if err := rows.Scan(&record.ID, &record.SiteID, &record.Timestamp, &record.StatusCode,
			&record.IsUp, &record.LatencyMs, &record.ErrorMessage); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// GetLatest gets the most recent record for a site
func (r *UptimeRepository) GetLatest(siteID int) (*UptimeRecord, error) {
	var record UptimeRecord
	err := r.db.QueryRow(`
		SELECT id, site_id, timestamp, status_code, is_up, latency_ms, error_message
		FROM uptime_records
		WHERE site_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`, siteID).Scan(&record.ID, &record.SiteID, &record.Timestamp, &record.StatusCode,
		&record.IsUp, &record.LatencyMs, &record.ErrorMessage)

	if err != nil {
		return nil, err
	}

	return &record, nil
}

// Count returns the total number of uptime records
func (r *UptimeRepository) Count(siteID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM uptime_records WHERE site_id = ?", siteID).Scan(&count)
	return count, err
}

// CountAll returns total uptime records across all sites
func (r *UptimeRepository) CountAll() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM uptime_records").Scan(&count)
	return count, err
}

// CountByDate counts records for a specific date
func (r *UptimeRepository) CountByDate(siteID int, date time.Time) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM uptime_records
		WHERE site_id = ? AND timestamp BETWEEN ? AND ?
	`, siteID, startOfDay, endOfDay).Scan(&count)

	return count, err
}

// CountSuccessByDate counts successful checks for a specific date
func (r *UptimeRepository) CountSuccessByDate(siteID int, date time.Time) (int, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM uptime_records
		WHERE site_id = ? AND is_up = 1 AND timestamp BETWEEN ? AND ?
	`, siteID, startOfDay, endOfDay).Scan(&count)

	return count, err
}

// GetAverageLatencyByDate gets average latency for a specific date
func (r *UptimeRepository) GetAverageLatencyByDate(siteID int, date time.Time) (float64, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var avgLatency float64
	err := r.db.QueryRow(`
		SELECT COALESCE(AVG(latency_ms), 0) FROM uptime_records
		WHERE site_id = ? AND timestamp BETWEEN ? AND ?
	`, siteID, startOfDay, endOfDay).Scan(&avgLatency)

	return avgLatency, err
}

// GetMinMaxLatencyByDate gets min and max latency for a specific date
func (r *UptimeRepository) GetMinMaxLatencyByDate(siteID int, date time.Time) (int64, int64, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	var minLatency, maxLatency int64
	err := r.db.QueryRow(`
		SELECT COALESCE(MIN(latency_ms), 0), COALESCE(MAX(latency_ms), 0) FROM uptime_records
		WHERE site_id = ? AND timestamp BETWEEN ? AND ?
	`, siteID, startOfDay, endOfDay).Scan(&minLatency, &maxLatency)

	return minLatency, maxLatency, err
}

// DeleteOlderThan deletes records older than the specified duration
func (r *UptimeRepository) DeleteOlderThan(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result, err := r.db.Exec(`
		DELETE FROM uptime_records WHERE timestamp < ?
	`, cutoffTime)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("✅ Deleted %d old uptime records", rowsAffected)
	}

	return rowsAffected, nil
}

// GetLastNDays retrieves records for the last N days
func (r *UptimeRepository) GetLastNDays(siteID int, days int) ([]UptimeRecord, error) {
	startTime := time.Now().AddDate(0, 0, -days)

	return r.GetByDateRange(siteID, startTime, time.Now())
}

// GetLastNHours retrieves records for the last N hours
func (r *UptimeRepository) GetLastNHours(siteID int, hours int) ([]UptimeRecord, error) {
	startTime := time.Now().Add(-time.Hour * time.Duration(hours))

	return r.GetByDateRange(siteID, startTime, time.Now())
}
// GetLatencyStats retrieves latency statistics for a time period
type LatencyStats struct {
	Time       time.Time `json:"time"`
	AvgLatency float64   `json:"avg_latency_ms"`
	MinLatency int64     `json:"min_latency_ms"`
	MaxLatency int64     `json:"max_latency_ms"`
}

func (r *UptimeRepository) GetLatencyStats(siteID int, startTime, endTime time.Time) ([]LatencyStats, error) {
	rows, err := r.db.Query(`
		SELECT 
			datetime(timestamp, 'start of hour') as hour,
			AVG(latency_ms) as avg,
			MIN(latency_ms) as min,
			MAX(latency_ms) as max
		FROM uptime_records
		WHERE site_id = ? AND timestamp >= ? AND timestamp <= ? AND latency_ms > 0
		GROUP BY hour
		ORDER BY hour ASC
	`, siteID, startTime, endTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []LatencyStats
	for rows.Next() {
		var s LatencyStats
		var timeStr string
		err := rows.Scan(&timeStr, &s.AvgLatency, &s.MinLatency, &s.MaxLatency)
		if err != nil {
			return nil, err
		}

		s.Time, _ = time.Parse("2006-01-02 15:04:05", timeStr)
		stats = append(stats, s)
	}

	return stats, rows.Err()
}