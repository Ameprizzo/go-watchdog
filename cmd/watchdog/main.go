package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/monitor"
	"github.com/Ameprizzo/go-watchdog/internal/notifier"
	"github.com/Ameprizzo/go-watchdog/internal/types"
)

var currentTicker *time.Ticker

func main() {
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

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
