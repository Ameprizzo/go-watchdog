package database

import (
	"log"
	"time"
)

// UptimeSummaryRepository handles uptime summary operations (daily aggregation)
type UptimeSummaryRepository struct {
	db *Database
}

// NewUptimeSummaryRepository creates a new uptime summary repository
func NewUptimeSummaryRepository(db *Database) *UptimeSummaryRepository {
	return &UptimeSummaryRepository{db: db}
}

// Create inserts or updates a daily uptime summary
func (r *UptimeSummaryRepository) Create(summary *UptimeSummary) error {
	// Check if summary exists for this date
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM uptime_summary WHERE site_id = ? AND date = ?)
	`, summary.SiteID, summary.Date).Scan(&exists)

	if err != nil {
		return err
	}

	if exists {
		// Update existing summary
		_, err = r.db.Exec(`
			UPDATE uptime_summary
			SET total_checks = ?, successful_checks = ?, failed_checks = ?,
			    uptime_percentage = ?, avg_latency_ms = ?, min_latency_ms = ?,
			    max_latency_ms = ?, downtime_minutes = ?, incident_count = ?,
			    updated_at = ?
			WHERE site_id = ? AND date = ?
		`, summary.TotalChecks, summary.SuccessfulChecks, summary.FailedChecks,
			summary.UptimePercentage, summary.AvgLatencyMs, summary.MinLatencyMs,
			summary.MaxLatencyMs, summary.DowntimeMinutes, summary.IncidentCount,
			time.Now(), summary.SiteID, summary.Date)

		return err
	}

	// Insert new summary
	result, err := r.db.Exec(`
		INSERT INTO uptime_summary
		(site_id, date, total_checks, successful_checks, failed_checks, uptime_percentage,
		 avg_latency_ms, min_latency_ms, max_latency_ms, downtime_minutes, incident_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, summary.SiteID, summary.Date, summary.TotalChecks, summary.SuccessfulChecks,
		summary.FailedChecks, summary.UptimePercentage, summary.AvgLatencyMs,
		summary.MinLatencyMs, summary.MaxLatencyMs, summary.DowntimeMinutes,
		summary.IncidentCount, time.Now(), time.Now())

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	summary.ID = int(id)
	log.Printf("✅ Uptime summary created for site %d on %s (%.2f%% uptime)",
		summary.SiteID, summary.Date.Format("2006-01-02"), summary.UptimePercentage)
	return nil
}

// GetByDate retrieves a summary for a specific date and site
func (r *UptimeSummaryRepository) GetByDate(siteID int, date time.Time) (*UptimeSummary, error) {
	dateStr := date.Format("2006-01-02")
	var summary UptimeSummary
	err := r.db.QueryRow(`
		SELECT id, site_id, date, total_checks, successful_checks, failed_checks, uptime_percentage,
		        avg_latency_ms, min_latency_ms, max_latency_ms, downtime_minutes, incident_count, created_at, updated_at
		FROM uptime_summary
		WHERE site_id = ? AND date = ?
	`, siteID, dateStr).Scan(&summary.ID, &summary.SiteID, &summary.Date, &summary.TotalChecks,
		&summary.SuccessfulChecks, &summary.FailedChecks, &summary.UptimePercentage,
		&summary.AvgLatencyMs, &summary.MinLatencyMs, &summary.MaxLatencyMs,
		&summary.DowntimeMinutes, &summary.IncidentCount, &summary.CreatedAt, &summary.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetBySiteIDAndDateRange retrieves summaries for a site within a date range
func (r *UptimeSummaryRepository) GetBySiteIDAndDateRange(siteID int, startDate, endDate time.Time) ([]UptimeSummary, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	rows, err := r.db.Query(`
		SELECT id, site_id, date, total_checks, successful_checks, failed_checks, uptime_percentage,
		        avg_latency_ms, min_latency_ms, max_latency_ms, downtime_minutes, incident_count, created_at, updated_at
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
		ORDER BY date DESC
	`, siteID, startStr, endStr)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []UptimeSummary
	for rows.Next() {
		var summary UptimeSummary
		if err := rows.Scan(&summary.ID, &summary.SiteID, &summary.Date, &summary.TotalChecks,
			&summary.SuccessfulChecks, &summary.FailedChecks, &summary.UptimePercentage,
			&summary.AvgLatencyMs, &summary.MinLatencyMs, &summary.MaxLatencyMs,
			&summary.DowntimeMinutes, &summary.IncidentCount, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, rows.Err()
}

// GetMonthlyAverage calculates the average uptime for a month
func (r *UptimeSummaryRepository) GetMonthlyAverage(siteID int, year, month int) (float64, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	var avgUptime float64
	err := r.db.QueryRow(`
		SELECT AVG(uptime_percentage)
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
	`, siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(&avgUptime)

	if err != nil {
		return 0, err
	}

	return avgUptime, nil
}

// GetWeeklyAverage calculates the average uptime for the past week
func (r *UptimeSummaryRepository) GetWeeklyAverage(siteID int) (float64, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	var avgUptime float64
	err := r.db.QueryRow(`
		SELECT AVG(uptime_percentage)
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
	`, siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(&avgUptime)

	if err != nil {
		return 0, err
	}

	return avgUptime, nil
}

// GetLastNDays retrieves summaries for the last N days
func (r *UptimeSummaryRepository) GetLastNDays(siteID int, days int) ([]UptimeSummary, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	return r.GetBySiteIDAndDateRange(siteID, startDate, endDate)
}

// GetTotalDowntimeInRange calculates total downtime minutes in a date range
func (r *UptimeSummaryRepository) GetTotalDowntimeInRange(siteID int, startDate, endDate time.Time) (int, error) {
	var totalDowntime int
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(downtime_minutes), 0)
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
	`, siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(&totalDowntime)

	return totalDowntime, err
}

// GetAverageLatencyInRange calculates average latency across summaries
func (r *UptimeSummaryRepository) GetAverageLatencyInRange(siteID int, startDate, endDate time.Time) (float64, error) {
	var avgLatency float64
	err := r.db.QueryRow(`
		SELECT AVG(avg_latency_ms)
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
	`, siteID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).Scan(&avgLatency)

	return avgLatency, err
}

// GetMaxIncidentsDay returns the date with the most incidents
func (r *UptimeSummaryRepository) GetMaxIncidentsDay(siteID int, startDate, endDate time.Time) (*UptimeSummary, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var summary UptimeSummary
	err := r.db.QueryRow(`
		SELECT id, site_id, date, total_checks, successful_checks, failed_checks, uptime_percentage,
		        avg_latency_ms, min_latency_ms, max_latency_ms, downtime_minutes, incident_count, created_at, updated_at
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
		ORDER BY incident_count DESC
		LIMIT 1
	`, siteID, startStr, endStr).Scan(&summary.ID, &summary.SiteID, &summary.Date, &summary.TotalChecks,
		&summary.SuccessfulChecks, &summary.FailedChecks, &summary.UptimePercentage,
		&summary.AvgLatencyMs, &summary.MinLatencyMs, &summary.MaxLatencyMs,
		&summary.DowntimeMinutes, &summary.IncidentCount, &summary.CreatedAt, &summary.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// GetSLAMet checks if site meets SLA target in a period
func (r *UptimeSummaryRepository) GetSLAMet(siteID int, slaTarget float64, startDate, endDate time.Time) (bool, float64, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var avgUptime float64
	err := r.db.QueryRow(`
		SELECT AVG(uptime_percentage)
		FROM uptime_summary
		WHERE site_id = ? AND date BETWEEN ? AND ?
	`, siteID, startStr, endStr).Scan(&avgUptime)

	if err != nil {
		return false, 0, err
	}

	return avgUptime >= slaTarget, avgUptime, nil
}

// Delete deletes a summary by ID
func (r *UptimeSummaryRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM uptime_summary WHERE id = ?", id)
	return err
}

// DeleteOlderThan deletes summaries older than specified days
func (r *UptimeSummaryRepository) DeleteOlderThan(days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.db.Exec(`
		DELETE FROM uptime_summary WHERE date < ?
	`, cutoffDate.Format("2006-01-02"))

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("✅ Deleted %d old uptime summaries", rowsAffected)
	}

	return rowsAffected, nil
}
