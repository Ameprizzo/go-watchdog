package database

import (
	"database/sql"
	"log"
	"time"
)

// IncidentRepository handles downtime incident operations
type IncidentRepository struct {
	db *Database
}

// NewIncidentRepository creates a new incident repository
func NewIncidentRepository(db *Database) *IncidentRepository {
	return &IncidentRepository{db: db}
}

// Create inserts a new incident
func (r *IncidentRepository) Create(incident *DowntimeIncident) error {
	result, err := r.db.Exec(`
		INSERT INTO downtime_incidents (site_id, start_time, end_time, duration_seconds, incident_count, root_cause)
		VALUES (?, ?, ?, ?, ?, ?)
	`, incident.SiteID, incident.StartTime, incident.EndTime, incident.DurationSeconds,
		incident.IncidentCount, incident.RootCause)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	incident.ID = int(id)
	log.Printf("✅ Incident created for site %d starting at %s", incident.SiteID, incident.StartTime.Format(time.RFC3339))
	return nil
}

// Update updates an existing incident
func (r *IncidentRepository) Update(incident *DowntimeIncident) error {
	var duration int64
	if incident.EndTime != nil {
		duration = int64(incident.EndTime.Sub(incident.StartTime).Seconds())
	}

	_, err := r.db.Exec(`
		UPDATE downtime_incidents 
		SET end_time = ?, duration_seconds = ?, incident_count = ?, root_cause = ?, updated_at = ?
		WHERE id = ?
	`, incident.EndTime, duration, incident.IncidentCount, incident.RootCause, time.Now(), incident.ID)

	if err != nil {
		return err
	}

	log.Printf("✅ Incident %d updated", incident.ID)
	return nil
}

// GetBySiteID retrieves all incidents for a site
func (r *IncidentRepository) GetBySiteID(siteID int) ([]DowntimeIncident, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, start_time, end_time, duration_seconds, incident_count, root_cause
		FROM downtime_incidents
		WHERE site_id = ?
		ORDER BY start_time DESC
	`, siteID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []DowntimeIncident
	for rows.Next() {
		var incident DowntimeIncident
		if err := rows.Scan(&incident.ID, &incident.SiteID, &incident.StartTime, &incident.EndTime,
			&incident.DurationSeconds, &incident.IncidentCount, &incident.RootCause); err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

// GetOngoing retrieves currently ongoing incidents (no end_time)
func (r *IncidentRepository) GetOngoing(siteID int) (*DowntimeIncident, error) {
	var incident DowntimeIncident
	err := r.db.QueryRow(`
		SELECT id, site_id, start_time, end_time, duration_seconds, incident_count, root_cause
		FROM downtime_incidents
		WHERE site_id = ? AND end_time IS NULL
		ORDER BY start_time DESC
		LIMIT 1
	`, siteID).Scan(&incident.ID, &incident.SiteID, &incident.StartTime, &incident.EndTime,
		&incident.DurationSeconds, &incident.IncidentCount, &incident.RootCause)

	if err != nil {
		if err == sql.ErrNoRows {
			// No ongoing incident - this is expected and not an error
			return nil, nil
		}
		return nil, err
	}

	return &incident, nil
}

// GetByDateRange retrieves incidents within a date range
func (r *IncidentRepository) GetByDateRange(siteID int, startTime, endTime time.Time) ([]DowntimeIncident, error) {
	rows, err := r.db.Query(`
		SELECT id, site_id, start_time, end_time, duration_seconds, incident_count, root_cause
		FROM downtime_incidents
		WHERE site_id = ? AND start_time BETWEEN ? AND ?
		ORDER BY start_time DESC
	`, siteID, startTime, endTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []DowntimeIncident
	for rows.Next() {
		var incident DowntimeIncident
		if err := rows.Scan(&incident.ID, &incident.SiteID, &incident.StartTime, &incident.EndTime,
			&incident.DurationSeconds, &incident.IncidentCount, &incident.RootCause); err != nil {
			return nil, err
		}
		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

// GetTotalDowntimeSince gets total downtime duration since a given time
func (r *IncidentRepository) GetTotalDowntimeSince(siteID int, since time.Time) (int64, error) {
	var totalSeconds int64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(duration_seconds), 0) FROM downtime_incidents
		WHERE site_id = ? AND start_time >= ?
	`, siteID, since).Scan(&totalSeconds)

	return totalSeconds, err
}

// GetIncidentCount returns the number of incidents for a site
func (r *IncidentRepository) GetIncidentCount(siteID int) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM downtime_incidents
		WHERE site_id = ? AND end_time IS NOT NULL
	`, siteID).Scan(&count)

	return count, err
}

// Count returns total incidents across all sites
func (r *IncidentRepository) Count(siteID int) (int, error) {
	if siteID == 0 {
		var count int
		err := r.db.QueryRow("SELECT COUNT(*) FROM downtime_incidents").Scan(&count)
		return count, err
	}
	return r.GetIncidentCount(siteID)
}

// GetLongestIncident retrieves the longest downtime incident
func (r *IncidentRepository) GetLongestIncident(siteID int) (*DowntimeIncident, error) {
	var incident DowntimeIncident
	err := r.db.QueryRow(`
		SELECT id, site_id, start_time, end_time, duration_seconds, incident_count, root_cause
		FROM downtime_incidents
		WHERE site_id = ? AND end_time IS NOT NULL
		ORDER BY duration_seconds DESC
		LIMIT 1
	`, siteID).Scan(&incident.ID, &incident.SiteID, &incident.StartTime, &incident.EndTime,
		&incident.DurationSeconds, &incident.IncidentCount, &incident.RootCause)

	if err != nil {
		return nil, err
	}

	return &incident, nil
}

// GetAverageIncidentDuration gets average downtime duration
func (r *IncidentRepository) GetAverageIncidentDuration(siteID int) (float64, error) {
	var avgDuration float64
	err := r.db.QueryRow(`
		SELECT COALESCE(AVG(duration_seconds), 0) FROM downtime_incidents
		WHERE site_id = ? AND end_time IS NOT NULL
	`, siteID).Scan(&avgDuration)

	return avgDuration, err
}

// Delete removes an incident
func (r *IncidentRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM downtime_incidents WHERE id = ?", id)
	return err
}

// DeleteOlderThan deletes incidents older than specified days
func (r *IncidentRepository) DeleteOlderThan(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result, err := r.db.Exec(`
		DELETE FROM downtime_incidents WHERE start_time < ?
	`, cutoffTime)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("✅ Deleted %d old incidents", rowsAffected)
	}

	return rowsAffected, nil
}
