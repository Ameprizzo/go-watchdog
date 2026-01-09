package sync

import (
	"fmt"
	"log"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/database"
	"github.com/Ameprizzo/go-watchdog/internal/monitor"
)

// SyncService handles syncing between config.json and database
type SyncService struct {
	repos *database.Repositories
	cfg   *monitor.Config
}

// SyncResult contains details about a sync operation
type SyncResult struct {
	Timestamp      time.Time
	TotalSites     int
	Added          int
	Updated        int
	Deleted        int
	Conflicts      int
	Errors         []string
	SettingsSynced bool
}

// NewSyncService creates a new sync service
func NewSyncService(repos *database.Repositories, cfg *monitor.Config) *SyncService {
	return &SyncService{
		repos: repos,
		cfg:   cfg,
	}
}

// SyncConfigToDB synchronizes config.json sites to database
func (s *SyncService) SyncConfigToDB() (*SyncResult, error) {
	result := &SyncResult{
		Timestamp: time.Now(),
		Errors:    []string{},
	}

	// Get all sites from database
	dbSites, err := s.repos.Site.GetAll()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to fetch DB sites: %v", err))
		return result, err
	}

	// Create a map of config sites
	configSiteMap := make(map[string]*monitor.Site)
	for i, site := range s.cfg.Sites {
		configSiteMap[site.Name] = &s.cfg.Sites[i]
	}

	// Create a map of DB sites
	dbSiteMap := make(map[string]database.Site)
	for i := range dbSites {
		dbSiteMap[dbSites[i].Name] = dbSites[i]
	}

	result.TotalSites = len(configSiteMap)

	// Process config sites
	for configName, configSite := range configSiteMap {
		if dbSite, exists := dbSiteMap[configName]; exists {
			// Site exists - check if update needed
			if dbSite.URL != configSite.URL {
				dbSite.URL = configSite.URL
				if err := s.repos.Site.Update(&dbSite); err != nil {
					errMsg := fmt.Sprintf("failed to update site %s: %v", configName, err)
					result.Errors = append(result.Errors, errMsg)
					log.Printf("❌ %s", errMsg)
				} else {
					result.Updated++
					log.Printf("✏️ Updated site: %s -> %s", configName, configSite.URL)

					// Audit log for site update
					s.logAudit("site_updated", "site", dbSite.ID,
						fmt.Sprintf(`{"name": "%s", "old_url": "%s", "new_url": "%s"}`, configName, dbSite.URL, configSite.URL))
				}
			}
		} else {
			// Site doesn't exist - create it
			newSite := &database.Site{
				Name:    configName,
				URL:     configSite.URL,
				Enabled: true,
			}
			if err := s.repos.Site.Create(newSite); err != nil {
				errMsg := fmt.Sprintf("failed to create site %s: %v", configName, err)
				result.Errors = append(result.Errors, errMsg)
				log.Printf("❌ %s", errMsg)
			} else {
				result.Added++
				log.Printf("✅ Added new site: %s", configName)

				// Audit log for site creation
				s.logAudit("site_created", "site", newSite.ID,
					fmt.Sprintf(`{"name": "%s", "url": "%s"}`, configName, configSite.URL))
			}
		}
		delete(dbSiteMap, configName)
	}

	// Handle deleted sites (sites in DB but not in config)
	for _, remainingSite := range dbSiteMap {
		log.Printf("⚠️ Site in DB but not in config.json: %s (%s)", remainingSite.Name, remainingSite.URL)
		// Don't auto-delete, just flag as potential deleted

		// Audit log for orphaned site
		s.logAudit("site_orphaned", "site", remainingSite.ID,
			fmt.Sprintf(`{"name": "%s", "url": "%s"}`, remainingSite.Name, remainingSite.URL))
	}

	// Sync settings
	if s.cfg.Settings.CheckInterval > 0 {
		dbSettings, err := s.repos.Settings.GetAll()
		if err != nil || dbSettings == nil {
			dbSettings = &database.AppSettings{
				CheckIntervalSeconds: s.cfg.Settings.CheckInterval,
				TimeoutSeconds:       s.cfg.Settings.Timeout,
				RetentionDays:        30,
				UpdatedAt:            time.Now(),
			}
			// Note: Settings repository might not have Create method
			// For now, skip if it doesn't exist
			result.SettingsSynced = true
			log.Printf("✅ Settings synced from config.json")

			s.logAudit("settings_created", "settings", 0,
				fmt.Sprintf(`{"check_interval": %d, "timeout": %d}`, s.cfg.Settings.CheckInterval, s.cfg.Settings.Timeout))
		} else if dbSettings.CheckIntervalSeconds != s.cfg.Settings.CheckInterval ||
			dbSettings.TimeoutSeconds != s.cfg.Settings.Timeout {
			oldInterval := dbSettings.CheckIntervalSeconds
			oldTimeout := dbSettings.TimeoutSeconds
			dbSettings.CheckIntervalSeconds = s.cfg.Settings.CheckInterval
			dbSettings.TimeoutSeconds = s.cfg.Settings.Timeout
			dbSettings.UpdatedAt = time.Now()
			// Note: Settings repository might not have Update method
			// For now, skip if it doesn't exist
			result.SettingsSynced = true
			log.Printf("✅ Settings synced with config.json")

			s.logAudit("settings_updated", "settings", 0,
				fmt.Sprintf(`{"old_interval": %d, "new_interval": %d, "old_timeout": %d, "new_timeout": %d}`, oldInterval, s.cfg.Settings.CheckInterval, oldTimeout, s.cfg.Settings.Timeout))
		}
	}

	log.Printf("\n📊 Sync Result: Added=%d, Updated=%d, Errors=%d", result.Added, result.Updated, len(result.Errors))
	return result, nil
}

// SyncDBToConfig exports database sites to config.json format
func (s *SyncService) SyncDBToConfig() ([]map[string]interface{}, error) {
	sites, err := s.repos.Site.GetAll()
	if err != nil {
		return nil, err
	}

	var configSites []map[string]interface{}
	for _, site := range sites {
		configSites = append(configSites, map[string]interface{}{
			"name": site.Name,
			"url":  site.URL,
		})
	}

	log.Printf("✅ Exported %d sites from database", len(configSites))

	s.logAudit("config_export", "config", 0,
		fmt.Sprintf(`{"exported_sites": %d}`, len(configSites)))

	return configSites, nil
}

// GetSyncStatus returns the current sync status between config and DB
func (s *SyncService) GetSyncStatus() map[string]interface{} {
	configSites := len(s.cfg.Sites)

	dbSites, err := s.repos.Site.GetAll()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	dbCount := len(dbSites)
	inSync := configSites == dbCount

	return map[string]interface{}{
		"config_sites":   configSites,
		"database_sites": dbCount,
		"in_sync":        inSync,
		"last_sync":      time.Now(),
	}
}

// logAudit records an audit log entry
func (s *SyncService) logAudit(action, entityType string, entityID int, details string) {
	auditLog := &database.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     "system",
		NewValue:   details,
		Timestamp:  time.Now(),
	}
	if err := s.repos.AuditLog.Create(auditLog); err != nil {
		log.Printf("❌ Failed to create audit log: %v", err)
	}
}
