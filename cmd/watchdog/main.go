package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/monitor"
)

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
			}{
				Results:         monitor.Store.Get(),
				RefreshInterval: cfg.Settings.CheckInterval,
				LastCheck:       time.Now().Format("15:04:05"),
			}

			tmpl.Execute(w, data)
		})

		// Manage page
		http.HandleFunc("/manage", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/manage.html"))
			tmpl.Execute(w, cfg)
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
					interval = 5 // Minimum 5 seconds
				}
				if timeout < 1 {
					timeout = 1 // Minimum 1 second
				}

				if err := cfg.UpdateSettings(interval, timeout); err != nil {
					log.Printf("Error updating settings: %v", err)
					http.Error(w, "Failed to update settings", http.StatusInternalServerError)
					return
				}

				// Restart the ticker with new interval
				go restartMonitoring(cfg)

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
	ticker := time.NewTicker(time.Duration(cfg.Settings.CheckInterval) * time.Second)
	defer ticker.Stop()

	fmt.Printf("🐕 Watchdog started. Checking %d sites every %ds...\n",
		len(cfg.Sites), cfg.Settings.CheckInterval)

	// Run immediately on start
	runChecks(cfg)

	// Keep the program running and check on every "tick"
	for range ticker.C {
		runChecks(cfg)
	}
}

func restartMonitoring(cfg *monitor.Config) {
	// Note: In production, you'd want proper ticker management
	// For now, this will create a new goroutine with updated interval
	go startMonitoring(cfg)
}

func runChecks(cfg *monitor.Config) {
	if len(cfg.Sites) == 0 {
		return
	}

	fmt.Printf("\n--- Check started at %s ---\n", time.Now().Format("15:04:05"))

	resultsChan := make(chan monitor.Result)
	var currentResults []monitor.Result

	for _, site := range cfg.Sites {
		go func(s monitor.Site) {
			resultsChan <- monitor.CheckSite(s, cfg.Settings.Timeout)
		}(site)
	}

	for i := 0; i < len(cfg.Sites); i++ {
		res := <-resultsChan
		currentResults = append(currentResults, res)

		icon := "✅"
		if !res.IsUp {
			icon = "❌"
		}
		fmt.Printf("%s %-20s | Latency: %v | Status: %d\n",
			icon, res.Name, res.Latency.Round(time.Millisecond), res.StatusCode)
	}

	// Update the global store for the Web UI
	monitor.Store.Update(currentResults)
}
