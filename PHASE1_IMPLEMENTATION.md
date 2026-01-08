# Phase 1 Implementation: Database Foundation

## ✅ Completed

### 1. Database Package Structure
- `internal/database/db.go` - SQLite connection, initialization, and migrations
- `internal/database/models.go` - All data structure definitions
- `internal/database/repositories/site_repo.go` - Site CRUD operations
- `internal/database/repositories/settings_repo.go` - Settings management
- `internal/database/repositories/repositories.go` - Repository factory

### 2. Database Schema Created
- **sites** table - Monitored websites
- **app_settings** table - Application-wide settings
- **uptime_records** table - Individual check results
- **uptime_summary** table - Daily aggregated statistics
- **downtime_incidents** table - Downtime tracking
- **notification_logs** table - Sent notifications history
- **audit_logs** table - User actions and changes

### 3. Key Features

#### Database Initialization
```go
// Open database connection
db, err := database.Open("watchdog.db")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Automatic migrations
// - Creates all tables
// - Creates indexes
// - Initializes default settings
```

#### Site Repository Operations
```go
repos := repositories.New(db)

// Create site
site := &database.Site{Name: "Google", URL: "https://google.com", Enabled: true}
repos.Sites.Create(site)

// Get all sites
sites, _ := repos.Sites.GetAll()

// Update site status
repos.Sites.UpdateStatusByName("Google", "up", time.Now())

// Delete site
repos.Sites.Delete(site.ID)
```

#### Settings Repository Operations
```go
// Get all settings
settings, _ := repos.Settings.GetAll()

// Update individual settings
repos.Settings.UpdateCheckInterval(20)
repos.Settings.UpdateTimeout(10)

// Update multiple settings
settings.Theme = "light"
repos.Settings.UpdateSettings(settings)
```

### 4. Database Features
- ✅ Foreign key constraints
- ✅ Automatic timestamps
- ✅ Indexed queries for performance
- ✅ Connection pooling
- ✅ Thread-safe operations with mutexes
- ✅ Transaction support

## 📋 Next Steps (Phase 2)

1. Integrate database with existing monitoring system
2. Record uptime checks to database
3. Implement daily aggregation job
4. Add incident tracking
5. Implement data retention cleanup

## 🔧 Installation

Add to `go.mod`:
```bash
go get github.com/mattn/go-sqlite3
```

## 📚 Usage Example

```go
package main

import (
    "log"
    "github.com/Ameprizzo/go-watchdog/internal/database"
    "github.com/Ameprizzo/go-watchdog/internal/database/repositories"
)

func main() {
    // Initialize database
    db, err := database.Open("watchdog.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get repositories
    repos := repositories.New(db)

    // Get all settings
    settings, err := repos.Settings.GetAll()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Check interval: %d seconds", settings.CheckIntervalSeconds)

    // Get all sites
    sites, err := repos.Sites.GetAll()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Monitoring %d sites", len(sites))
}
```

## 📊 Database Schema

```sql
-- Sites
sites: id | name | url | enabled | current_status | last_checked | created_at | updated_at

-- Settings
app_settings: id | check_interval | timeout | retention_days | sla_target | enable_history | theme | timezone

-- Uptime Records
uptime_records: id | site_id | timestamp | status_code | is_up | latency_ms | error_message

-- Daily Summary
uptime_summary: id | site_id | date | total_checks | successful_checks | uptime_percentage | latency_stats

-- Incidents
downtime_incidents: id | site_id | start_time | end_time | duration | incident_count

-- Logs
notification_logs: id | notification_id | site_id | type | message | severity | sent_at | status
audit_logs: id | action | entity_type | entity_id | old_value | new_value | timestamp
```

## 🎯 Data Persistence Achieved

Phase 1 establishes:
- ✅ Persistent site storage
- ✅ Persistent settings storage
- ✅ Database foundation for historical data
- ✅ Tables for all future data needs
- ✅ Automatic schema migrations

The foundation is ready for Phase 2 integration with the existing monitoring system.
