package database

import (
	"log"
	"time"
)

// AnalyticsService provides high-level analytics and aggregation operations
type AnalyticsService struct {
	repos *Repositories
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(repos *Repositories) *AnalyticsService {
	return &AnalyticsService{repos: repos}
}

// GenerateDailySummary calculates and stores daily uptime summary for a site
func (s *AnalyticsService) GenerateDailySummary(siteID int, summaryDate time.Time) error {
	// Get all uptime records for the date
	startTime := time.Date(summaryDate.Year(), summaryDate.Month(), summaryDate.Day(), 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 1).Add(-time.Nanosecond)

	records, err := s.repos.Uptime.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		log.Printf("⚠️  No uptime records found for site %d on %s", siteID, summaryDate.Format("2006-01-02"))
		return nil
	}

	// Calculate metrics
	totalChecks := len(records)
	successfulChecks := 0
	var totalLatency int64
	minLatency := int64(^uint64(0) >> 1) // Max int64
	maxLatency := int64(0)

	for _, record := range records {
		if record.IsUp && record.StatusCode >= 200 && record.StatusCode < 300 {
			successfulChecks++
		}

		if record.LatencyMs > 0 {
			totalLatency += record.LatencyMs
			if record.LatencyMs < minLatency {
				minLatency = record.LatencyMs
			}
			if record.LatencyMs > maxLatency {
				maxLatency = record.LatencyMs
			}
		}
	}

	failedChecks := totalChecks - successfulChecks
	uptimePercentage := (float64(successfulChecks) / float64(totalChecks)) * 100

	var avgLatency float64
	if totalChecks > 0 {
		avgLatency = float64(totalLatency) / float64(totalChecks)
	}

	if minLatency == int64(^uint64(0)>>1) {
		minLatency = 0
	}

	// Get downtime incidents and minutes for this day
	incidents, err := s.repos.Incident.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return err
	}

	incidentCount := len(incidents)
	var downtimeMinutes int64

	for _, incident := range incidents {
		if incident.DurationSeconds > 0 {
			downtimeMinutes += incident.DurationSeconds / 60
		}
	}

	// Create or update summary
	summary := &UptimeSummary{
		SiteID:           siteID,
		Date:             summaryDate,
		TotalChecks:      totalChecks,
		SuccessfulChecks: successfulChecks,
		FailedChecks:     failedChecks,
		UptimePercentage: uptimePercentage,
		AvgLatencyMs:     avgLatency,
		MinLatencyMs:     minLatency,
		MaxLatencyMs:     maxLatency,
		DowntimeMinutes:  downtimeMinutes,
		IncidentCount:    incidentCount,
	}

	return s.repos.UptimeSummary.Create(summary)
}

// GenerateDailySummariesForAll generates summaries for all sites on a given date
func (s *AnalyticsService) GenerateDailySummariesForAll(summaryDate time.Time) error {
	sites, err := s.repos.Site.GetAll()
	if err != nil {
		return err
	}

	for _, site := range sites {
		if err := s.GenerateDailySummary(site.ID, summaryDate); err != nil {
			log.Printf("❌ Error generating daily summary for site %d: %v", site.ID, err)
			// Continue with other sites instead of failing completely
		}
	}

	log.Printf("✅ Daily summaries generated for %d sites", len(sites))
	return nil
}

// GetUptimePercentage calculates uptime percentage for a period
func (s *AnalyticsService) GetUptimePercentage(siteID int, startTime, endTime time.Time) (float64, error) {
	records, err := s.repos.Uptime.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	successfulChecks := 0
	for _, record := range records {
		if record.IsUp && record.StatusCode >= 200 && record.StatusCode < 300 {
			successfulChecks++
		}
	}

	return (float64(successfulChecks) / float64(len(records))) * 100, nil
}

// GetAverageLatency calculates average latency for a period
func (s *AnalyticsService) GetAverageLatency(siteID int, startTime, endTime time.Time) (float64, error) {
	records, err := s.repos.Uptime.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	var totalLatency int64
	count := 0

	for _, record := range records {
		if record.LatencyMs > 0 {
			totalLatency += record.LatencyMs
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	return float64(totalLatency) / float64(count), nil
}

// CheckSLACompliance checks if a site meets SLA target over a period
func (s *AnalyticsService) CheckSLACompliance(siteID int, slaTarget float64, startTime, endTime time.Time) (bool, float64, error) {
	actualUptime, err := s.GetUptimePercentage(siteID, startTime, endTime)
	if err != nil {
		return false, 0, err
	}

	return actualUptime >= slaTarget, actualUptime, nil
}

// GetLongestIncident finds the longest downtime incident in a period
func (s *AnalyticsService) GetLongestIncident(siteID int, startTime, endTime time.Time) (*DowntimeIncident, error) {
	incidents, err := s.repos.Incident.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}

	if len(incidents) == 0 {
		return nil, nil
	}

	longest := incidents[0]
	for _, incident := range incidents[1:] {
		if incident.DurationSeconds > longest.DurationSeconds {
			longest = incident
		}
	}

	return &longest, nil
}

// GetMTTR calculates mean time to recovery (average incident duration)
func (s *AnalyticsService) GetMTTR(siteID int, startTime, endTime time.Time) (int64, error) {
	incidents, err := s.repos.Incident.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return 0, err
	}

	if len(incidents) == 0 {
		return 0, nil
	}

	var totalDuration int64
	for _, incident := range incidents {
		if incident.EndTime != nil {
			totalDuration += incident.DurationSeconds
		}
	}

	return totalDuration / int64(len(incidents)), nil
}

// GetTotalDowntime calculates total downtime in a period
func (s *AnalyticsService) GetTotalDowntime(siteID int, startTime, endTime time.Time) (int64, error) {
	incidents, err := s.repos.Incident.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return 0, err
	}

	var totalDowntime int64
	for _, incident := range incidents {
		totalDowntime += incident.DurationSeconds
	}

	return totalDowntime, nil
}

// CleanupOldData removes old records based on retention policy
func (s *AnalyticsService) CleanupOldData(retentionDays int) error {
	log.Printf("🧹 Starting data cleanup (retention: %d days)...", retentionDays)

	// Delete old uptime records
	deletedUptime, err := s.repos.Uptime.DeleteOlderThan(retentionDays)
	if err != nil {
		return err
	}

	// Delete old incidents
	deletedIncidents, err := s.repos.Incident.DeleteOlderThan(retentionDays)
	if err != nil {
		return err
	}

	// Delete old summaries
	deletedSummaries, err := s.repos.UptimeSummary.DeleteOlderThan(retentionDays)
	if err != nil {
		return err
	}

	// Delete old notifications
	deletedNotifications, err := s.repos.Notification.DeleteOlderThan(retentionDays)
	if err != nil {
		return err
	}

	// Delete old audit logs
	deletedAuditLogs, err := s.repos.AuditLog.DeleteOlderThan(retentionDays)
	if err != nil {
		return err
	}

	log.Printf("✅ Cleanup complete: %d uptime records, %d incidents, %d summaries, %d notifications, %d audit logs deleted",
		deletedUptime, deletedIncidents, deletedSummaries, deletedNotifications, deletedAuditLogs)

	return nil
}

// GetSiteMetrics retrieves comprehensive metrics for a site
func (s *AnalyticsService) GetSiteMetrics(siteID int, days int) (map[string]interface{}, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	metrics := make(map[string]interface{})

	// Uptime percentage
	uptime, err := s.GetUptimePercentage(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	metrics["uptime_percentage"] = uptime

	// Average latency
	avgLatency, err := s.GetAverageLatency(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	metrics["avg_latency_ms"] = avgLatency

	// Total downtime
	totalDowntime, err := s.GetTotalDowntime(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	metrics["total_downtime_seconds"] = totalDowntime

	// Incident count
	incidents, err := s.repos.Incident.GetByDateRange(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	metrics["incident_count"] = len(incidents)

	// MTTR
	if len(incidents) > 0 {
		mttr, err := s.GetMTTR(siteID, startTime, endTime)
		if err != nil {
			return nil, err
		}
		metrics["mttr_seconds"] = mttr
	}

	// Longest incident
	longest, err := s.GetLongestIncident(siteID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if longest != nil {
		metrics["longest_incident_seconds"] = longest.DurationSeconds
	}

	metrics["period_days"] = days
	metrics["start_time"] = startTime
	metrics["end_time"] = endTime

	return metrics, nil
}

// AnalyticsRepository handles analytics queries
type AnalyticsRepository struct {
	db *Database
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *Database) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

// SiteMetrics represents comprehensive metrics for a single site
type SiteMetrics struct {
	SiteID               int        `json:"site_id"`
	SiteName             string     `json:"site_name"`
	CurrentStatus        string     `json:"current_status"`
	LastChecked          *time.Time `json:"last_checked"`
	UptimePercentage     float64    `json:"uptime_percentage"`
	DowntimePercentage   float64    `json:"downtime_percentage"`
	AvgLatency           float64    `json:"avg_latency_ms"`
	MinLatency           int64      `json:"min_latency_ms"`
	MaxLatency           int64      `json:"max_latency_ms"`
	TotalIncidents       int        `json:"total_incidents"`
	ActiveIncidents      int        `json:"active_incidents"`
	TotalChecks          int        `json:"total_checks"`
	SuccessfulChecks     int        `json:"successful_checks"`
	FailedChecks         int        `json:"failed_checks"`
	TotalDowntimeSeconds int64      `json:"total_downtime_seconds"`
	LastIncidentTime     *time.Time `json:"last_incident_time"`
}

// DashboardSummary represents overall dashboard statistics
type DashboardSummary struct {
	TotalSites          int       `json:"total_sites"`
	SitesUp             int       `json:"sites_up"`
	SitesDown           int       `json:"sites_down"`
	AverageUptime       float64   `json:"average_uptime"`
	TotalIncidentsLast7 int       `json:"total_incidents_last_7"`
	SitesWithIssues     int       `json:"sites_with_issues"`
	LastUpdate          time.Time `json:"last_update"`
}

// SLAReport represents SLA compliance data
type SLAReport struct {
	SiteID              int     `json:"site_id"`
	SiteName            string  `json:"site_name"`
	SLATargetPercentage float64 `json:"sla_target_percentage"`
	ActualUptime        float64 `json:"actual_uptime"`
	SLACompliant        bool    `json:"sla_compliant"`
	UptimeGap           float64 `json:"uptime_gap"`
}

// GetDashboardSummary retrieves overall dashboard metrics
func (r *AnalyticsRepository) GetDashboardSummary() (*DashboardSummary, error) {
	var summary DashboardSummary
	summary.LastUpdate = time.Now()

	// Get site counts
	err := r.db.QueryRow(`
		SELECT COUNT(*) as total,
		       SUM(CASE WHEN current_status = 'up' THEN 1 ELSE 0 END) as up,
		       SUM(CASE WHEN current_status = 'down' THEN 1 ELSE 0 END) as down
		FROM sites WHERE enabled = 1
	`).Scan(&summary.TotalSites, &summary.SitesUp, &summary.SitesDown)

	if err != nil {
		return nil, err
	}

	// Get average uptime
	err = r.db.QueryRow(`
		SELECT AVG(uptime_percentage)
		FROM uptime_summary
		WHERE date >= date('now', '-7 days')
	`).Scan(&summary.AverageUptime)

	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, err
	}

	// Get recent incidents
	err = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM downtime_incidents
		WHERE start_time >= datetime('now', '-7 days')
	`).Scan(&summary.TotalIncidentsLast7)

	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, err
	}

	// Get sites with issues (active incidents)
	err = r.db.QueryRow(`
		SELECT COUNT(DISTINCT site_id)
		FROM downtime_incidents
		WHERE end_time IS NULL
	`).Scan(&summary.SitesWithIssues)

	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, err
	}

	return &summary, nil
}

// GetSLAReport retrieves SLA compliance for all sites
func (r *AnalyticsRepository) GetSLAReport(days int) ([]SLAReport, error) {
	startTime := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.Query(`
		SELECT 
			s.id,
			s.name,
			COALESCE((SELECT AVG(uptime_percentage) FROM uptime_summary 
			 WHERE site_id = s.id AND date >= ?), 0) as uptime,
			(SELECT sla_target_percentage FROM app_settings LIMIT 1) as sla
		FROM sites s
		WHERE s.enabled = 1
		ORDER BY s.name
	`, startTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []SLAReport
	for rows.Next() {
		var report SLAReport
		var sla *float64

		err := rows.Scan(&report.SiteID, &report.SiteName, &report.ActualUptime, &sla)
		if err != nil {
			return nil, err
		}

		if sla != nil {
			report.SLATargetPercentage = *sla
		}

		report.SLACompliant = report.ActualUptime >= report.SLATargetPercentage
		report.UptimeGap = report.SLATargetPercentage - report.ActualUptime

		reports = append(reports, report)
	}

	return reports, rows.Err()
}
