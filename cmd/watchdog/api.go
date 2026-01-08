package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/database"
)

// setupAPIRoutes configures all API endpoints
func setupAPIRoutes(repos *database.Repositories) {
	// Analytics endpoints
	http.HandleFunc("/api/analytics/dashboard-summary", func(w http.ResponseWriter, r *http.Request) {
		handleDashboardSummary(w, r, repos)
	})

	http.HandleFunc("/api/analytics/site-metrics", func(w http.ResponseWriter, r *http.Request) {
		handleSiteMetrics(w, r, repos)
	})

	http.HandleFunc("/api/analytics/uptime-trend", func(w http.ResponseWriter, r *http.Request) {
		handleUptimeTrend(w, r, repos)
	})

	http.HandleFunc("/api/analytics/incidents", func(w http.ResponseWriter, r *http.Request) {
		handleIncidentReport(w, r, repos)
	})

	http.HandleFunc("/api/analytics/sla-report", func(w http.ResponseWriter, r *http.Request) {
		handleSLAReport(w, r, repos)
	})

	http.HandleFunc("/api/analytics/latency-stats", func(w http.ResponseWriter, r *http.Request) {
		handleLatencyStats(w, r, repos)
	})

	// Site endpoints with filtering
	http.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {
		handleGetSites(w, r, repos)
	})

	http.HandleFunc("/api/sites/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearchSites(w, r, repos)
	})

	// Reports endpoint
	http.HandleFunc("/api/reports/generate", func(w http.ResponseWriter, r *http.Request) {
		handleGenerateReport(w, r, repos)
	})
}

// handleDashboardSummary returns overall dashboard statistics
func handleDashboardSummary(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	summary, err := repos.Analytics.GetDashboardSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

// handleSiteMetrics returns metrics for a specific site
func handleSiteMetrics(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	siteID := r.URL.Query().Get("site_id")
	days := r.URL.Query().Get("days")

	if siteID == "" {
		http.Error(w, "missing site_id parameter", http.StatusBadRequest)
		return
	}

	siteIDInt, err := strconv.Atoi(siteID)
	if err != nil {
		http.Error(w, "invalid site_id", http.StatusBadRequest)
		return
	}

	daysInt := 30
	if days != "" {
		d, err := strconv.Atoi(days)
		if err == nil {
			daysInt = d
		}
	}

	analytics := database.NewAnalyticsService(repos)
	metrics, err := analytics.GetSiteMetrics(siteIDInt, daysInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(metrics)
}

// handleUptimeTrend returns uptime trend data for a site
func handleUptimeTrend(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	siteID := r.URL.Query().Get("site_id")
	days := r.URL.Query().Get("days")

	if siteID == "" {
		http.Error(w, "missing site_id parameter", http.StatusBadRequest)
		return
	}

	siteIDInt, err := strconv.Atoi(siteID)
	if err != nil {
		http.Error(w, "invalid site_id", http.StatusBadRequest)
		return
	}

	daysInt := 30
	if days != "" {
		d, err := strconv.Atoi(days)
		if err == nil {
			daysInt = d
		}
	}

	startTime := time.Now().AddDate(0, 0, -daysInt)
	trends, err := repos.UptimeSummary.GetBySiteIDAndDateRange(siteIDInt, startTime, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(trends)
}

// handleIncidentReport returns incident data for a period
func handleIncidentReport(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	siteID := r.URL.Query().Get("site_id")
	days := r.URL.Query().Get("days")

	if siteID == "" {
		http.Error(w, "missing site_id parameter", http.StatusBadRequest)
		return
	}

	siteIDInt, err := strconv.Atoi(siteID)
	if err != nil {
		http.Error(w, "invalid site_id", http.StatusBadRequest)
		return
	}

	daysInt := 30
	if days != "" {
		d, err := strconv.Atoi(days)
		if err == nil {
			daysInt = d
		}
	}

	startTime := time.Now().AddDate(0, 0, -daysInt)
	incidents, err := repos.Incident.GetByDateRange(siteIDInt, startTime, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(incidents)
}

// handleSLAReport returns SLA compliance report
func handleSLAReport(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	days := r.URL.Query().Get("days")
	daysInt := 30

	if days != "" {
		d, err := strconv.Atoi(days)
		if err == nil {
			daysInt = d
		}
	}

	reports, err := repos.Analytics.GetSLAReport(daysInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(reports)
}

// handleLatencyStats returns latency statistics
func handleLatencyStats(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	siteID := r.URL.Query().Get("site_id")
	hours := r.URL.Query().Get("hours")

	if siteID == "" {
		http.Error(w, "missing site_id parameter", http.StatusBadRequest)
		return
	}

	siteIDInt, err := strconv.Atoi(siteID)
	if err != nil {
		http.Error(w, "invalid site_id", http.StatusBadRequest)
		return
	}

	hoursInt := 24
	if hours != "" {
		h, err := strconv.Atoi(hours)
		if err == nil {
			hoursInt = h
		}
	}

	startTime := time.Now().Add(-time.Duration(hoursInt) * time.Hour)
	stats, err := repos.Uptime.GetLatencyStats(siteIDInt, startTime, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

// handleGetSites returns all sites with optional filtering
func handleGetSites(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	status := r.URL.Query().Get("status")
	enabled := r.URL.Query().Get("enabled")

	sites, err := repos.Site.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by status
	if status != "" {
		var filtered []database.Site
		for _, site := range sites {
			if site.CurrentStatus == status {
				filtered = append(filtered, site)
			}
		}
		sites = filtered
	}

	// Filter by enabled
	if enabled != "" {
		var filtered []database.Site
		enabledBool := enabled == "true"
		for _, site := range sites {
			if site.Enabled == enabledBool {
				filtered = append(filtered, site)
			}
		}
		sites = filtered
	}

	json.NewEncoder(w).Encode(sites)
}

// handleSearchSites searches sites by name
func handleSearchSites(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter", http.StatusBadRequest)
		return
	}

	sites, err := repos.Site.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []database.Site
	for _, site := range sites {
		// Simple substring search
		if site.Name == query || site.URL == query {
			results = append(results, site)
		}
	}

	json.NewEncoder(w).Encode(results)
}

// ReportRequest represents a report generation request
type ReportRequest struct {
	ReportType string `json:"report_type"` // "sla", "incidents", "uptime"
	SiteID     int    `json:"site_id"`
	Days       int    `json:"days"`
	Format     string `json:"format"` // "json", "csv"
}

// handleGenerateReport generates various reports
func handleGenerateReport(w http.ResponseWriter, r *http.Request, repos *database.Repositories) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch req.ReportType {
	case "sla":
		reports, err := repos.Analytics.GetSLAReport(req.Days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(reports)

	case "incidents":
		startTime := time.Now().AddDate(0, 0, -req.Days)
		incidents, err := repos.Incident.GetByDateRange(req.SiteID, startTime, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(incidents)

	case "uptime":
		analytics := database.NewAnalyticsService(repos)
		metrics, err := analytics.GetSiteMetrics(req.SiteID, req.Days)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(metrics)

	default:
		http.Error(w, "unknown report type", http.StatusBadRequest)
	}
}
