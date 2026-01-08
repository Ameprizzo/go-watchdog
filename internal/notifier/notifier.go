package notifier

import (
	"fmt"
	"log"
	"net/http"
	"bytes"
	"encoding/json"
)

// Alert sends a notification to the console and/or webhooks
func Alert(siteName string, url string, isUp bool) {
	status := "UP ✅"
	if !isUp {
		status = "DOWN ❌"
	}

	// 1. Console Logging (Always)
	message := fmt.Sprintf("ALERT: [%s] is %s (URL: %s)", siteName, status, url)
	log.Println(message)

	// 2. Optional: Discord Webhook (Uncomment and add URL to test)
	// sendDiscordWebhook("YOUR_WEBHOOK_URL_HERE", message)
}

func sendDiscordWebhook(webhookURL string, message string) {
	payload := map[string]string{"content": message}
	jsonBody, _ := json.Marshal(payload)
	
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Printf("Failed to send webhook: %v", err)
		return
	}
	defer resp.Body.Close()
}