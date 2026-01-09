package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/backup"
	"github.com/Ameprizzo/go-watchdog/internal/database"
	"github.com/Ameprizzo/go-watchdog/internal/monitor"
	"github.com/Ameprizzo/go-watchdog/internal/notifier"
	"github.com/Ameprizzo/go-watchdog/internal/sync"
	"github.com/Ameprizzo/go-watchdog/internal/types"
)

var currentTicker *time.Ticker
var globalRepos *database.Repositories
var globalMaintenance *database.MaintenanceService
var globalSyncService *sync.SyncService
var globalBackupService *backup.BackupService
var backupTicker *time.Ticker

func main() {
	// Initialize database
	db, err := database.Open("watchdog.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get repositories
	repos := database.NewRepositories(db)
	globalRepos = repos

	// Load config (existing code)
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Load settings from database if available
	dbSettings, err := repos.Settings.GetAll()
	if err == nil && dbSettings != nil {
		cfg.Settings.CheckInterval = dbSettings.CheckIntervalSeconds
		cfg.Settings.Timeout = dbSettings.TimeoutSeconds
	}

	// Sync sites from config.json to database
	for _, site := range cfg.Sites {
		exists, _ := repos.Site.Exists(site.Name)
		if !exists {
			dbSite := &database.Site{
				Name:    site.Name,
				URL:     site.URL,
				Enabled: true,
			}
			repos.Site.Create(dbSite)
		}
	}

	// Initialize and start maintenance service (daily aggregation and cleanup)
	retentionDays := 30
	if dbSettings != nil {
		retentionDays = dbSettings.RetentionDays
	}
	globalMaintenance = database.NewMaintenanceService(repos, retentionDays)
	globalMaintenance.Start()
	defer globalMaintenance.Stop()

	// Initialize notifier with repositories for database persistence
	notifier.StatusTracker.SetRepositories(repos)
	// Create a mapping of site names to IDs for the notifier
	siteIDMap := make(map[string]int)
	for _, site := range cfg.Sites {
		dbSite, _ := repos.Site.GetByName(site.Name)
		if dbSite != nil {
			siteIDMap[site.Name] = dbSite.ID
		}
	}
	notifier.StatusTracker.SetSiteIDMap(siteIDMap)

	// Initialize sync service (Week 5)
	globalSyncService = sync.NewSyncService(repos, cfg)
	fmt.Println("🔄 Sync service initialized")

	// Initialize backup service (Week 5)
	backupSvc, err := backup.NewBackupService(repos, "backups", 10)
	if err != nil {
		log.Fatalf("Failed to initialize backup service: %v", err)
	}
	globalBackupService = backupSvc

	// Start periodic backups (daily at 2 AM)
	go startPeriodicBackups()

	// Setup API routes (Phase 3)
	setupAPIRoutes(repos)

	// Setup Sync & Backup routes (Week 5)
	setupSyncAndBackupRoutes(repos)

	// Start the web server in the background
	go func() {
		// Dashboard handler with auto-refresh
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/index.html"))

			data := struct {
				Results         []monitor.Result
				RefreshInterval int
				LastCheck       string
				Notifications   []notifier.DashboardNotification
				UnreadCount     int
			}{
				Results:         monitor.Store.Get(),
				RefreshInterval: cfg.Settings.CheckInterval,
				LastCheck:       time.Now().Format("15:04:05"),
				Notifications:   notifier.Notifications.GetAll(),
				UnreadCount:     len(notifier.Notifications.GetUnread()),
			}

			tmpl.Execute(w, data)
		})

		// API endpoint to get notifications as JSON
		http.HandleFunc("/api/notifications", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(notifier.Notifications.GetAll())
		})

		// API endpoint to get unread count
		http.HandleFunc("/api/notifications/unread-count", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]int{
				"count": len(notifier.Notifications.GetUnread()),
			})
		})

		// Mark notification as read
		http.HandleFunc("/api/notifications/mark-read", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				id := r.FormValue("id")
				notifier.Notifications.MarkAsRead(id)
				w.WriteHeader(http.StatusOK)
			}
		})

		// Mark all notifications as read
		http.HandleFunc("/api/notifications/mark-all-read", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				notifier.Notifications.MarkAllAsRead()
				w.WriteHeader(http.StatusOK)
			}
		})

		// Clear all notifications
		http.HandleFunc("/api/notifications/clear", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				notifier.Notifications.Clear()
				w.WriteHeader(http.StatusOK)
			}
		})

		// Manage page
		http.HandleFunc("/manage", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/manage.html"))
			tmpl.Execute(w, cfg)
		})

		// Notifications settings page
		http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/notifications.html"))
			tmpl.Execute(w, cfg)
		})

		// Add notification channel
		http.HandleFunc("/notifications/add", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				channelType := types.NotificationChannel(r.FormValue("type"))
				enabled := r.FormValue("enabled") == "on"

				settings := make(map[string]string)
				// Parse settings based on channel type
				switch channelType {
				case types.ChannelEmail:
					settings["smtp_host"] = r.FormValue("smtp_host")
					settings["smtp_port"] = r.FormValue("smtp_port")
					settings["username"] = r.FormValue("username")
					settings["password"] = r.FormValue("password")
					settings["from"] = r.FormValue("from")
					settings["to"] = r.FormValue("to")
				case types.ChannelDiscord:
					settings["webhook_url"] = r.FormValue("webhook_url")
				case types.ChannelTelegram:
					settings["bot_token"] = r.FormValue("bot_token")
					settings["chat_id"] = r.FormValue("chat_id")
				case types.ChannelSlack:
					settings["webhook_url"] = r.FormValue("webhook_url")
				}

				channel := types.ChannelSettings{
					Type:     channelType,
					Enabled:  enabled,
					Settings: settings,
				}

				if err := cfg.AddNotificationChannel(channel); err != nil {
					log.Printf("Error adding notification channel: %v", err)
					http.Error(w, "Failed to add notification channel", http.StatusInternalServerError)
					return
				}

				http.Redirect(w, r, "/notifications", http.StatusSeeOther)
			}
		})

		// Toggle notification channel
		http.HandleFunc("/notifications/toggle", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				channelType := types.NotificationChannel(r.FormValue("type"))

				for i, ch := range cfg.Notifications.Channels {
					if ch.Type == channelType {
						cfg.Notifications.Channels[i].Enabled = !cfg.Notifications.Channels[i].Enabled
						cfg.UpdateNotificationSettings(cfg.Notifications)
						break
					}
				}

				http.Redirect(w, r, "/notifications", http.StatusSeeOther)
			}
		})

		// Delete notification channel
		http.HandleFunc("/notifications/delete", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				channelType := types.NotificationChannel(r.FormValue("type"))
				if err := cfg.RemoveNotificationChannel(channelType); err != nil {
					log.Printf("Error removing notification channel: %v", err)
					http.Error(w, "Failed to remove notification channel", http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, "/notifications", http.StatusSeeOther)
			}
		})

		// Add site (redirects to manage page after adding)
		http.HandleFunc("/settings/add", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				name := r.FormValue("name")
				url := r.FormValue("url")

				newSite := monitor.Site{Name: name, URL: url}
				if err := cfg.AddSite(newSite); err != nil {
					log.Printf("Error adding site: %v", err)
					http.Error(w, "Failed to add site", http.StatusInternalServerError)
					return
				}

				http.Redirect(w, r, "/manage", http.StatusSeeOther)
			}
		})

		// Delete site
		http.HandleFunc("/manage/delete", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				name := r.FormValue("name")
				if err := cfg.DeleteSite(name); err != nil {
					log.Printf("Error deleting site: %v", err)
					http.Error(w, "Failed to delete site", http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, "/manage", http.StatusSeeOther)
			}
		})

		// Update global settings
		http.HandleFunc("/manage/settings", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				interval, _ := strconv.Atoi(r.FormValue("interval"))
				timeout, _ := strconv.Atoi(r.FormValue("timeout"))

				if interval < 5 {
					interval = 5
				}
				if timeout < 1 {
					timeout = 1
				}

				if err := cfg.UpdateSettings(interval, timeout); err != nil {
					log.Printf("Error updating settings: %v", err)
					http.Error(w, "Failed to update settings", http.StatusInternalServerError)
					return
				}

				// Restart monitoring with new interval
				restartMonitoring(cfg)

				http.Redirect(w, r, "/manage", http.StatusSeeOther)
			}
		})

		// Trigger daily summary generation
		http.HandleFunc("/api/admin/generate-summary", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if globalRepos == nil {
				http.Error(w, "Database not initialized", http.StatusInternalServerError)
				return
			}

			analytics := database.NewAnalyticsService(globalRepos)
			yesterday := time.Now().AddDate(0, 0, -1)

			if err := analytics.GenerateDailySummariesForAll(yesterday); err != nil {
				http.Error(w, fmt.Sprintf("Failed to generate summaries: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Daily summaries generated for yesterday"})
		})

		// Get site analytics
		http.HandleFunc("/api/admin/analytics", func(w http.ResponseWriter, r *http.Request) {
			if globalRepos == nil {
				http.Error(w, "Database not initialized", http.StatusInternalServerError)
				return
			}

			siteIDStr := r.URL.Query().Get("site_id")
			if siteIDStr == "" {
				http.Error(w, "site_id parameter required", http.StatusBadRequest)
				return
			}

			siteID, err := strconv.Atoi(siteIDStr)
			if err != nil {
				http.Error(w, "Invalid site_id", http.StatusBadRequest)
				return
			}

			analytics := database.NewAnalyticsService(globalRepos)
			metrics, err := analytics.GetSiteMetrics(siteID, 30)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(metrics)
		})

		// Get database statistics
		http.HandleFunc("/api/admin/stats", func(w http.ResponseWriter, r *http.Request) {
			if globalRepos == nil {
				http.Error(w, "Database not initialized", http.StatusInternalServerError)
				return
			}

			uptimeCount, _ := globalRepos.Uptime.CountAll()
			incidentCount, _ := globalRepos.Incident.Count(0)
			notificationCount, _ := globalRepos.Notification.Count(0)
			auditCount, _ := globalRepos.AuditLog.Count()

			stats := map[string]interface{}{
				"uptime_records":     uptimeCount,
				"incidents":          incidentCount,
				"notifications":      notificationCount,
				"audit_logs":         auditCount,
				"maintenance_status": globalMaintenance.GetStatus(),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats)
		})
		// Phase 4: UI Pages (Uptime Details and Reports)
		http.HandleFunc("/uptime-detail", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/uptime_detail.html"))
			tmpl.Execute(w, nil)
		})

		http.HandleFunc("/reports", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/reports.html"))
			tmpl.Execute(w, nil)
		})
		fmt.Println("🌐 Web dashboard running at http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Start monitoring
	startMonitoring(cfg)
}

func startMonitoring(cfg *monitor.Config) {
	if currentTicker != nil {
		currentTicker.Stop()
	}

	currentTicker = time.NewTicker(time.Duration(cfg.Settings.CheckInterval) * time.Second)

	fmt.Printf("🐕 Watchdog started. Checking %d sites every %ds...\n",
		len(cfg.Sites), cfg.Settings.CheckInterval)

	// Run immediately on start
	runChecks(cfg)

	// Keep the program running and check on every "tick"
	for range currentTicker.C {
		runChecks(cfg)
	}
}

func restartMonitoring(cfg *monitor.Config) {
	go startMonitoring(cfg)
}

func runChecks(cfg *monitor.Config) {
	if len(cfg.Sites) == 0 {
		return
	}

	fmt.Printf("\n--- Check started at %s ---\n", time.Now().Format("15:04:05"))

	resultsChan := make(chan types.Result)
	var currentResults []types.Result

	for _, site := range cfg.Sites {
		go func(s monitor.Site) {
			resultsChan <- monitor.CheckSite(s, cfg.Settings.Timeout)
		}(site)
	}

	for i := 0; i < len(cfg.Sites); i++ {
		res := <-resultsChan
		currentResults = append(currentResults, res)

		notifier.StatusTracker.CheckAndNotify(res, cfg.Notifications)

		// Record uptime check to database
		if globalRepos != nil {
			// Look up the site ID by name
			site, err := globalRepos.Site.GetByName(res.Name)
			if err != nil {
				log.Printf("Error looking up site %s: %v", res.Name, err)
				goto printResult
			}

			// Populate the site ID in the result
			res.ID = site.ID
			currentResults[len(currentResults)-1] = res

			latencyMs := res.Latency.Milliseconds()
			uptimeRecord := &database.UptimeRecord{
				SiteID:       site.ID,
				Timestamp:    time.Now(),
				StatusCode:   res.StatusCode,
				IsUp:         res.IsUp,
				LatencyMs:    latencyMs,
				ErrorMessage: "",
			}

			if err := globalRepos.Uptime.Create(uptimeRecord); err != nil {
				log.Printf("Error recording uptime for %s: %v", res.Name, err)
			}

			// Handle incident tracking
			if !res.IsUp {
				// Check if there's an ongoing incident
				ongoing, err := globalRepos.Incident.GetOngoing(site.ID)
				if err != nil {
					log.Printf("Error checking ongoing incident for %s: %v", res.Name, err)
				} else if ongoing == nil {
					// No ongoing incident, create a new one
					incident := &database.DowntimeIncident{
						SiteID:    site.ID,
						StartTime: time.Now(),
					}
					if err := globalRepos.Incident.Create(incident); err != nil {
						log.Printf("❌ Error creating incident for %s (ID: %d): %v", res.Name, site.ID, err)
					} else {
						log.Printf("🚨 Incident started for %s (ID: %d)", res.Name, site.ID)

						// Create audit log for incident creation
						auditLog := &database.AuditLog{
							Action:     "incident_started",
							EntityType: "incident",
							EntityID:   incident.ID,
							UserID:     "system",
							NewValue:   fmt.Sprintf(`{"site_id": %d, "site_name": "%s", "start_time": "%s"}`, site.ID, res.Name, incident.StartTime.Format(time.RFC3339)),
							Timestamp:  time.Now(),
						}
						if err := globalRepos.AuditLog.Create(auditLog); err != nil {
							log.Printf("Error creating audit log for incident: %v", err)
						}
					}
				} else {
					log.Printf("⏳ Ongoing incident already exists for %s", res.Name)
				}
			} else {
				// Site is up, close any ongoing incidents
				ongoing, err := globalRepos.Incident.GetOngoing(site.ID)
				if err != nil {
					log.Printf("Error checking ongoing incident for %s: %v", res.Name, err)
				} else if ongoing != nil {
					now := time.Now()
					ongoing.EndTime = &now
					ongoing.DurationSeconds = int64(now.Sub(ongoing.StartTime).Seconds())

					if err := globalRepos.Incident.Update(ongoing); err != nil {
						log.Printf("❌ Error closing incident for %s: %v", res.Name, err)
					} else {
						log.Printf("✅ Incident closed for %s (duration: %d seconds)", res.Name, ongoing.DurationSeconds)

						// Create audit log for incident closure
						auditLog := &database.AuditLog{
							Action:     "incident_closed",
							EntityType: "incident",
							EntityID:   ongoing.ID,
							UserID:     "system",
							NewValue:   fmt.Sprintf(`{"site_id": %d, "site_name": "%s", "start_time": "%s", "end_time": "%s", "duration_seconds": %d}`, site.ID, res.Name, ongoing.StartTime.Format(time.RFC3339), ongoing.EndTime.Format(time.RFC3339), ongoing.DurationSeconds),
							Timestamp:  time.Now(),
						}
						if err := globalRepos.AuditLog.Create(auditLog); err != nil {
							log.Printf("Error creating audit log for incident closure: %v", err)
						}
					}
				}
			}
		}

	printResult:
		icon := "✅"
		if !res.IsUp {
			icon = "❌"
		}
		fmt.Printf("%s %-20s | Latency: %v | Status: %d\n",
			icon, res.Name, res.Latency.Round(time.Millisecond), res.StatusCode)
	}

	// Convert []types.Result to []monitor.Result
	monitorResults := make([]monitor.Result, len(currentResults))
	for i, r := range currentResults {
		monitorResults[i] = monitor.Result(r)
	}

	// Update the global store for the Web UI
	monitor.Store.Update(monitorResults)
}

// startPeriodicBackups runs database backups on a schedule
func startPeriodicBackups() {
	if globalBackupService == nil {
		log.Printf("❌ Backup service not initialized")
		return
	}

	// Run first backup immediately
	result, err := globalBackupService.CreateBackup()
	if err != nil {
		log.Printf("❌ Initial backup failed: %v", err)
	} else if result.Success {
		log.Printf("✅ Initial backup created: %s (%.2f MB)", result.BackupFile, float64(result.CompressedSize)/1024/1024)
	}

	// Schedule daily backups
	backupTicker = time.NewTicker(24 * time.Hour)
	defer backupTicker.Stop()

	for range backupTicker.C {
		log.Printf("⏱️ Running scheduled daily backup...")
		result, err := globalBackupService.CreateBackup()
		if err != nil {
			log.Printf("❌ Scheduled backup failed: %v", err)
		} else if result.Success {
			log.Printf("✅ Scheduled backup created: %s (%.2f MB)", result.BackupFile, float64(result.CompressedSize)/1024/1024)
		}
	}
}
