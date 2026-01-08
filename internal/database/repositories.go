package database

// Repositories holds all database repositories for centralized access
type Repositories struct {
	Site          *SiteRepository
	Settings      *SettingsRepository
	Uptime        *UptimeRepository
	Incident      *IncidentRepository
	Notification  *NotificationRepository
	AuditLog      *AuditLogRepository
	UptimeSummary *UptimeSummaryRepository
	Analytics     *AnalyticsRepository
}

// NewRepositories creates and initializes all repositories
func NewRepositories(db *Database) *Repositories {
	return &Repositories{
		Site:          NewSiteRepository(db),
		Settings:      NewSettingsRepository(db),
		Uptime:        NewUptimeRepository(db),
		Incident:      NewIncidentRepository(db),
		Notification:  NewNotificationRepository(db),
		AuditLog:      NewAuditLogRepository(db),
		UptimeSummary: NewUptimeSummaryRepository(db),
		Analytics:     NewAnalyticsRepository(db),
	}
}
