package main

import (
	"fmt"
	"log"

	"github.com/Ameprizzo/go-watchdog/internal/monitor"
)

func main() {
	fmt.Println("🐕 Go-Watchdog is starting...")

	//Load configuration
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Print loaded SITES for verification
	fmt.Printf("Monitoring %d sites:\n", len(cfg.Sites))
	for _, site := range cfg.Sites {
		fmt.Printf("- %s (%s)\n", site.Name, site.URL)
	}

}
