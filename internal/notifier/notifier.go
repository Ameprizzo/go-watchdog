package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"sync"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/database"
	"github.com/Ameprizzo/go-watchdog/internal/types"
)

// DashboardNotification represents an in-app notification
type DashboardNotification struct {
	ID        string    `json:"id"`
	SiteName  string    `json:"site_name"`
	Message   string    `json:"message"`
	Severity  string    `json:"severity"` // "error", "warning", "info", "success"
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}

// NotificationStore manages dashboard notifications
type NotificationStore struct {
	mu            sync.RWMutex
	Notifications []DashboardNotification
	maxSize       int
}

var Notifications = &NotificationStore{
	maxSize: 100, // Keep last 100 notifications
}

// Add adds a new notification to the store
func (ns *NotificationStore) Add(notification DashboardNotification) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Add to beginning of slice
	ns.Notifications = append([]DashboardNotification{notification}, ns.Notifications...)

	// Trim if exceeds max size
	if len(ns.Notifications) > ns.maxSize {
		ns.Notifications = ns.Notifications[:ns.maxSize]
	}
}

// GetAll returns all notifications
func (ns *NotificationStore) GetAll() []DashboardNotification {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.Notifications
}

// GetUnread returns only unread notifications
func (ns *NotificationStore) GetUnread() []DashboardNotification {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	var unread []DashboardNotification
	for _, n := range ns.Notifications {
		if !n.Read {
			unread = append(unread, n)
		}
	}
	return unread
}

// MarkAsRead marks a notification as read
func (ns *NotificationStore) MarkAsRead(id string) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for i := range ns.Notifications {
		if ns.Notifications[i].ID == id {
			ns.Notifications[i].Read = true
			break
		}
	}
}

// MarkAllAsRead marks all notifications as read
func (ns *NotificationStore) MarkAllAsRead() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for i := range ns.Notifications {
		ns.Notifications[i].Read = true
	}
}

// Clear removes all notifications
func (ns *NotificationStore) Clear() {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.Notifications = []DashboardNotification{}
}

// SiteStatusTracker tracks site status to detect changes
type SiteStatusTracker struct {
	mu            sync.RWMutex
	previousState map[string]bool // siteName -> isUp
	repos         *database.Repositories
	siteIDMap     map[string]int // siteName -> siteID
}

func NewStatusTracker() *SiteStatusTracker {
	return &SiteStatusTracker{
		previousState: make(map[string]bool),
		siteIDMap:     make(map[string]int),
	}
}

var StatusTracker = NewStatusTracker()

// SetRepositories sets the database repositories for saving notifications and audit logs
func (st *SiteStatusTracker) SetRepositories(repos *database.Repositories) {
	st.repos = repos
}

// SetSiteIDMap sets the mapping of site names to IDs
func (st *SiteStatusTracker) SetSiteIDMap(siteIDMap map[string]int) {
	st.siteIDMap = siteIDMap
}

// CheckAndNotify checks if status changed and sends notifications
func (st *SiteStatusTracker) CheckAndNotify(result types.Result, config *types.NotificationConfig) {
	st.mu.Lock()
	previousStatus, existed := st.previousState[result.Name]
	st.previousState[result.Name] = result.IsUp
	st.mu.Unlock()

	// If this is first check or status hasn't changed, don't notify
	if !existed || previousStatus == result.IsUp {
		return
	}

	// Status changed - send notifications
	if !result.IsUp {
		// Site went down
		st.sendNotifications(result, "down", config)
	} else {
		// Site came back up
		st.sendNotifications(result, "up", config)
	}
}

func (st *SiteStatusTracker) sendNotifications(result types.Result, status string, config *types.NotificationConfig) {
	if config == nil || !config.Enabled {
		return
	}

	var message string
	var severity string

	if status == "down" {
		message = fmt.Sprintf("🔴 Site DOWN: %s (%s) - Status: %d", result.Name, result.URL, result.StatusCode)
		severity = "error"
	} else {
		message = fmt.Sprintf("🟢 Site RECOVERED: %s (%s) is back online", result.Name, result.URL)
		severity = "success"
	}

	// Send to each enabled channel
	for _, channel := range config.Channels {
		if !channel.Enabled {
			continue
		}

		switch channel.Type {
		case types.ChannelDashboard:
			st.sendDashboardNotification(result, message, severity)
		case types.ChannelEmail:
			go st.sendEmailNotification(result, message, channel.Settings)
		case types.ChannelDiscord:
			go st.sendDiscordNotification(result, message, status, channel.Settings)
		case types.ChannelTelegram:
			go st.sendTelegramNotification(result, message, channel.Settings)
		case types.ChannelSlack:
			go st.sendSlackNotification(result, message, status, channel.Settings)
		}
	}
}

// Dashboard notification
func (st *SiteStatusTracker) sendDashboardNotification(result types.Result, message, severity string) {
	notification := DashboardNotification{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		SiteName:  result.Name,
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Read:      false,
	}
	Notifications.Add(notification)
	log.Println("📱 Dashboard notification:", message)

	// Save to database if repositories are available
	if st.repos != nil {
		siteID := st.siteIDMap[result.Name]
		notifLog := &database.NotificationLog{
			NotificationID: notification.ID,
			SiteID:         siteID,
			Type:           "dashboard",
			Message:        message,
			Severity:       severity,
			SentAt:         notification.Timestamp,
			Status:         "sent",
			RetryCount:     0,
		}
		if err := st.repos.Notification.Create(notifLog); err != nil {
			log.Printf("Error saving notification log to database: %v", err)
		}
	}
}

// Email notification
func (st *SiteStatusTracker) sendEmailNotification(result types.Result, message string, settings map[string]string) {
	smtpHost := settings["smtp_host"]
	smtpPort := settings["smtp_port"]
	username := settings["username"]
	password := settings["password"]
	from := settings["from"]
	to := settings["to"]

	if smtpHost == "" || username == "" || password == "" || to == "" {
		log.Println("⚠️  Email notification skipped: missing configuration")
		return
	}

	subject := fmt.Sprintf("Alert: %s Status Change", result.Name)
	body := fmt.Sprintf("Subject: %s\r\n\r\n%s\r\n\r\nTimestamp: %s\r\nLatency: %v",
		subject, message, time.Now().Format(time.RFC1123), result.Latency)

	auth := smtp.PlainAuth("", username, password, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(body))
	if err != nil {
		log.Printf("❌ Email notification failed: %v\n", err)
	} else {
		log.Println("📧 Email notification sent")
	}
}

// Discord notification
func (st *SiteStatusTracker) sendDiscordNotification(result types.Result, message, status string, settings map[string]string) {
	webhookURL := settings["webhook_url"]
	if webhookURL == "" {
		log.Println("⚠️  Discord notification skipped: missing webhook URL")
		return
	}

	color := 15158332 // Red
	if status == "up" {
		color = 3066993 // Green
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("Site Status Alert: %s", result.Name),
				"description": message,
				"color":       color,
				"fields": []map[string]interface{}{
					{"name": "URL", "value": result.URL, "inline": true},
					{"name": "Status Code", "value": fmt.Sprintf("%d", result.StatusCode), "inline": true},
					{"name": "Latency", "value": result.Latency.String(), "inline": true},
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Discord notification failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("💬 Discord notification sent")
	} else {
		log.Printf("❌ Discord notification failed with status: %d\n", resp.StatusCode)
	}
}

// Telegram notification
func (st *SiteStatusTracker) sendTelegramNotification(result types.Result, message string, settings map[string]string) {
	botToken := settings["bot_token"]
	chatID := settings["chat_id"]

	if botToken == "" || chatID == "" {
		log.Println("⚠️  Telegram notification skipped: missing configuration")
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Telegram notification failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("✈️  Telegram notification sent")
	} else {
		log.Printf("❌ Telegram notification failed with status: %d\n", resp.StatusCode)
	}
}

// Slack notification
func (st *SiteStatusTracker) sendSlackNotification(result types.Result, message, status string, settings map[string]string) {
	webhookURL := settings["webhook_url"]
	if webhookURL == "" {
		log.Println("⚠️  Slack notification skipped: missing webhook URL")
		return
	}

	color := "danger"
	if status == "up" {
		color = "good"
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": fmt.Sprintf("Site Status Alert: %s", result.Name),
				"text":  message,
				"fields": []map[string]interface{}{
					{"title": "URL", "value": result.URL, "short": true},
					{"title": "Status Code", "value": fmt.Sprintf("%d", result.StatusCode), "short": true},
					{"title": "Latency", "value": result.Latency.String(), "short": true},
				},
				"footer": "Go-Watchdog",
				"ts":     time.Now().Unix(),
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Slack notification failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Println("💼 Slack notification sent")
	} else {
		log.Printf("❌ Slack notification failed with status: %d\n", resp.StatusCode)
	}
}
