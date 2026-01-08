package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/monitor"
)

func main() {
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

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

	results := make(chan monitor.Result)

	for _, site := range cfg.Sites {
		go func(s monitor.Site) {
			results <- monitor.CheckSite(s, cfg.Settings.Timeout)
		}(site)
	}

	for i := 0; i < len(cfg.Sites); i++ {
		res := <-results
		icon := "✅"
		if !res.IsUp {
			icon = "❌"
		}
		fmt.Printf("%s %-15s | Latency: %v | Code: %d\n",
			icon, res.Name, res.Latency.Round(time.Millisecond), res.StatusCode)
	}
}
