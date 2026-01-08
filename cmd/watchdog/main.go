package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/monitor"
)

func main() {
	fmt.Println("🐕 Go-Watchdog is starting...")

	//Load configuration
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create a channel to receive results
	results := make(chan monitor.Result)

	fmt.Println("🐕 Watchdog active. Starting checks...")

	// Launch a goroutine for each site
	for _, site := range cfg.Sites {
		fmt.Printf("- %s (%s)\n", site.Name, site.URL)
		go func(s monitor.Site) {
			result := monitor.CheckSite(s, cfg.Settings.Timeout)
			results <- result
		}(site)
	}

	//Collect and print results as they come in
	for i := 0; i < len(cfg.Sites); i++ {
		res := <-results
		status := "✅ UP"
		if !res.IsUp {
			status = "❌ DOWN"
		}
		fmt.Printf("[%s] %-15s | Latency: %v | Status: %d\n", status, res.Name, res.Latency.Round(time.Millisecond), res.StatusCode)

	}
}
