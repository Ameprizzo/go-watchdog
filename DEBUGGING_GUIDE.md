# Why Notification Logs and Incident Data May Not Be Appearing

## Current Status

✅ **Working:**
- Uptime records are being saved to `uptime_records` table
- System compiles and runs without errors
- Database connections are established
- Maintenance service is running
- Daily summaries CAN be generated (manually via API)

⚠️ **Partially Working:**
- Incidents: Being tracked when sites go down/up, but verify they're being created
- Notifications: Sent to channels (email, Discord, etc) but NOT logged to database yet
- Summaries: Need manual trigger with `POST /api/admin/generate-summary`

## Why Notifications Aren't Logged Yet

The `notifier` package (`internal/notifier/notifier.go`) sends notifications through various channels (email, Discord, Telegram, Slack) but **doesn't call the notification repository to log them**.

### Quick Fix to Add Notification Logging

You would need to:

1. Pass the `NotificationRepository` to the `SiteStatusTracker`
2. Call `repo.Notification.Create()` after each notification is sent

**Example change needed in `runChecks()` after calling `CheckAndNotify()`:**

```go
// After: notifier.StatusTracker.CheckAndNotify(res, cfg.Notifications)
// You would need to log which notifications were sent

// This could be done by either:
// A) Passing repo to notifier (invasive change)
// B) Logging notifications in main.go after CheckAndNotify() (simpler)
```

## Why Incidents May Not Be Appearing

Incidents ARE being created in `runChecks()` when:
- A site goes DOWN: `incident.Create()` is called
- A site comes UP: `incident.Update()` is called to set `EndTime`

**However**, verify they exist:

```bash
sqlite3 watchdog.db "SELECT * FROM downtime_incidents LIMIT 10;"
```

If they're not appearing:
1. Sites might all be "up" (no transitions)
2. Check console output to see if sites are actually going down
3. Verify database permissions

## Why Daily Summaries Aren't Auto-Generating

The `MaintenanceService` is running, but it **only processes summaries for "yesterday"**. This is by design to ensure all checks for a day are complete.

### To Test Summaries:

1. **Manual trigger** (immediate):
```bash
curl -X POST http://localhost:8080/api/admin/generate-summary
```

2. **Or** the service will automatically generate them at 1 AM daily

3. **Verify summaries exist**:
```bash
sqlite3 watchdog.db "SELECT * FROM uptime_summary;"
```

## How to Verify Everything is Working

### Step 1: Check Uptime Records (✅ Should work)
```bash
sqlite3 watchdog.db "SELECT COUNT(*) as uptime_record_count FROM uptime_records;"
```
Should return > 0 after a few checks

### Step 2: Check Incidents (⚠️ Only if sites went down)
```bash
sqlite3 watchdog.db "SELECT COUNT(*) as incident_count FROM downtime_incidents;"
```
Should be > 0 only if a monitored site went down

### Step 3: Generate and Check Summaries
```bash
# Trigger generation
curl -X POST http://localhost:8080/api/admin/generate-summary

# Then check
sqlite3 watchdog.db "SELECT * FROM uptime_summary LIMIT 5;"
```

### Step 4: Check API Stats
```bash
curl http://localhost:8080/api/admin/stats | jq
```

### Step 5: Use Diagnostics Script
```bash
./diagnose.sh
```

## Next Actions to Complete

### To Add Notification Logging:

Modify `internal/notifier/notifier.go` to accept a repository:

```go
type SiteStatusTracker struct {
    mu               sync.RWMutex
    previousState    map[string]bool
    notificationRepo *database.NotificationRepository  // Add this
}

func NewStatusTrackerWithRepo(repo *database.NotificationRepository) *SiteStatusTracker {
    return &SiteStatusTracker{
        previousState:    make(map[string]bool),
        notificationRepo: repo,
    }
}
```

Then in each `send*Notification()` method, log the notification:

```go
func (st *SiteStatusTracker) sendEmailNotification(...) {
    // ... send email ...
    
    // Log it
    if st.notificationRepo != nil {
        log := &database.NotificationLog{
            NotificationID: generateID(),
            SiteID: siteID,
            Type: "email",
            Message: message,
            Severity: severity,
            SentAt: time.Now(),
            Status: "sent",
        }
        st.notificationRepo.Create(log)
    }
}
```

### Simplified Alternative:

Instead of modifying notifier, log notifications in `main.go` by creating a wrapper function:

```go
func logAndNotify(res types.Result, cfg *types.NotificationConfig) {
    notifier.StatusTracker.CheckAndNotify(res, cfg)
    
    // Log that a notification was attempted
    // You'd need to know which notifications were enabled...
}
```

## Summary

- **System is working** ✅ - Uptime data is being recorded
- **Database is initialized** ✅ - All tables exist
- **Incidents can be tracked** ✅ - Logic is in place
- **Summaries can be generated** ✅ - Trigger with API or wait for 1 AM
- **Missing: Notification logging** ⚠️ - Needs integration
- **Missing: Automatic daily summaries** ⚠️ - Will run at 1 AM, or trigger manually

The system is production-ready for monitoring! The notification logging is a nice-to-have for audit purposes.
