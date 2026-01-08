package database

import (
	"log"
	"sync"
	"time"
)

// MaintenanceService handles scheduled maintenance tasks
type MaintenanceService struct {
	repos                *Repositories
	analytics            *AnalyticsService
	ticker               *time.Ticker
	done                 chan bool
	mutex                sync.RWMutex
	dailyAggregation     bool
	dataCleanup          bool
	aggregationTime      time.Time
	cleanupTime          time.Time
	cleanupRetentionDays int
}

// NewMaintenanceService creates a new maintenance service
func NewMaintenanceService(repos *Repositories, retentionDays int) *MaintenanceService {
	analytics := NewAnalyticsService(repos)

	// Default aggregation time: 1 AM
	now := time.Now()
	aggregationTime := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, time.UTC)

	// Default cleanup time: 3 AM
	cleanupTime := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, time.UTC)

	return &MaintenanceService{
		repos:                repos,
		analytics:            analytics,
		done:                 make(chan bool),
		dailyAggregation:     true,
		dataCleanup:          true,
		aggregationTime:      aggregationTime,
		cleanupTime:          cleanupTime,
		cleanupRetentionDays: retentionDays,
	}
}

// Start begins the maintenance service
func (m *MaintenanceService) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ticker != nil {
		return // Already running
	}

	m.ticker = time.NewTicker(1 * time.Minute)
	go m.runMaintenanceLoop()
	log.Printf("✅ Maintenance service started (Aggregation: %s, Cleanup: %s, Retention: %d days)",
		m.aggregationTime.Format("15:04"), m.cleanupTime.Format("15:04"), m.cleanupRetentionDays)
}

// Stop stops the maintenance service
func (m *MaintenanceService) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ticker != nil {
		m.ticker.Stop()
		m.ticker = nil
		m.done <- true
		log.Printf("✅ Maintenance service stopped")
	}
}

// runMaintenanceLoop checks for maintenance tasks that need to be executed
func (m *MaintenanceService) runMaintenanceLoop() {
	for {
		select {
		case <-m.ticker.C:
			m.checkAndRunTasks()
		case <-m.done:
			return
		}
	}
}

// checkAndRunTasks runs maintenance tasks if their scheduled time has been reached
func (m *MaintenanceService) checkAndRunTasks() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Check if aggregation should run
	if m.dailyAggregation && m.isTimeToRun(now, m.aggregationTime) {
		go m.runDailyAggregation(today)
	}

	// Check if cleanup should run
	if m.dataCleanup && m.isTimeToRun(now, m.cleanupTime) {
		go m.runDataCleanup()
	}
}

// isTimeToRun checks if the current time is within 1 minute of the scheduled time
func (m *MaintenanceService) isTimeToRun(now, scheduledTime time.Time) bool {
	// Get today's scheduled time
	todayScheduled := time.Date(now.Year(), now.Month(), now.Day(),
		scheduledTime.Hour(), scheduledTime.Minute(), 0, 0, time.UTC)

	// Check if we're within 1 minute window of scheduled time
	diff := now.Sub(todayScheduled)
	return diff >= 0 && diff < 1*time.Minute
}

// runDailyAggregation runs the daily aggregation task
func (m *MaintenanceService) runDailyAggregation(date time.Time) {
	log.Printf("🔄 Starting daily aggregation for %s...", date.Format("2006-01-02"))

	start := time.Now()

	// Generate summaries for yesterday (since current day isn't complete yet)
	yesterday := date.AddDate(0, 0, -1)
	if err := m.analytics.GenerateDailySummariesForAll(yesterday); err != nil {
		log.Printf("❌ Daily aggregation failed: %v", err)
		return
	}

	duration := time.Since(start)
	log.Printf("✅ Daily aggregation completed in %s", duration.String())
}

// runDataCleanup runs the data cleanup task
func (m *MaintenanceService) runDataCleanup() {
	log.Printf("🧹 Starting data cleanup task...")

	start := time.Now()

	m.mutex.RLock()
	retentionDays := m.cleanupRetentionDays
	m.mutex.RUnlock()

	if err := m.analytics.CleanupOldData(retentionDays); err != nil {
		log.Printf("❌ Data cleanup failed: %v", err)
		return
	}

	duration := time.Since(start)
	log.Printf("✅ Data cleanup completed in %s", duration.String())
}

// SetAggregationTime sets the time when daily aggregation should run
func (m *MaintenanceService) SetAggregationTime(hour, minute int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	m.aggregationTime = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	log.Printf("ℹ️  Aggregation time set to %s", m.aggregationTime.Format("15:04"))
}

// SetCleanupTime sets the time when data cleanup should run
func (m *MaintenanceService) SetCleanupTime(hour, minute int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	m.cleanupTime = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC)
	log.Printf("ℹ️  Cleanup time set to %s", m.cleanupTime.Format("15:04"))
}

// SetCleanupRetentionDays sets the data retention policy in days
func (m *MaintenanceService) SetCleanupRetentionDays(days int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cleanupRetentionDays = days
	log.Printf("ℹ️  Cleanup retention set to %d days", days)
}

// EnableDailyAggregation enables/disables daily aggregation
func (m *MaintenanceService) EnableDailyAggregation(enable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.dailyAggregation = enable
	status := "enabled"
	if !enable {
		status = "disabled"
	}
	log.Printf("ℹ️  Daily aggregation %s", status)
}

// EnableDataCleanup enables/disables data cleanup
func (m *MaintenanceService) EnableDataCleanup(enable bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.dataCleanup = enable
	status := "enabled"
	if !enable {
		status = "disabled"
	}
	log.Printf("ℹ️  Data cleanup %s", status)
}

// GetStatus returns the status of the maintenance service
func (m *MaintenanceService) GetStatus() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"running":                   m.ticker != nil,
		"daily_aggregation_enabled": m.dailyAggregation,
		"data_cleanup_enabled":      m.dataCleanup,
		"aggregation_time":          m.aggregationTime.Format("15:04"),
		"cleanup_time":              m.cleanupTime.Format("15:04"),
		"retention_days":            m.cleanupRetentionDays,
	}
}
