package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
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
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/index.html"))
			tmpl.Execute(w, monitor.Store.Get())
		})

		// 1. Show Settings Page
		http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("web/templates/settings.html"))
			tmpl.Execute(w, nil)
		})

		// 2. Handle Form Submission
		http.HandleFunc("/settings/add", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				name := r.FormValue("name")
				url := r.FormValue("url")

				newSite := monitor.Site{Name: name, URL: url}
				cfg.AddSite(newSite) // Update memory and JSON file

				http.Redirect(w, r, "/", http.StatusSeeOther)
			}
		})
		fmt.Println("🌐 Web dashboard running at http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Create a ticker based on our config interval
	ticker := time.NewTicker(time.Duration(cfg.Settings.CheckInterval) * time.Second)

	fmt.Printf("🐕 Watchdog started. Checking %d sites every %ds...\n",
		len(cfg.Sites), cfg.Settings.CheckInterval)

	// Run immediately on start
	runChecks(cfg)

	// Keep the program running and check on every "tick"
	for range ticker.C {
		runChecks(cfg)
	}
}

func runChecks(cfg *monitor.Config) {
	fmt.Printf("\n--- Check started at %s ---\n", time.Now().Format("15:04:05"))

	resultsChan := make(chan monitor.Result)
	var currentResults []monitor.Result // Temporary slice to hold this round's results

	for _, site := range cfg.Sites {
		go func(s monitor.Site) {
			resultsChan <- monitor.CheckSite(s, cfg.Settings.Timeout)
		}(site)
	}

	for i := 0; i < len(cfg.Sites); i++ {
		res := <-resultsChan
		currentResults = append(currentResults, res) // Save result to slice

		icon := "✅"
		if !res.IsUp {
			icon = "❌"
		}
		fmt.Printf("%s %-15s | Latency: %v\n", icon, res.Name, res.Latency.Round(time.Millisecond))
	}

	// KEY STEP: Push the slice of results to the global store for the Web UI
	monitor.Store.Update(currentResults)
}
