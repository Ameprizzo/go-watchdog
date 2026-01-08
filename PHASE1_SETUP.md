# Phase 1 Setup Guide

## Installation Steps

### 1. Add SQLite Dependency
Run this command in your project root:

```bash
go get github.com/mattn/go-sqlite3
```

### 2. Database Location
The database file will be created automatically at:
```
./watchdog.db
```

You can change this path when calling `database.Open()`:
```go
db, err := database.Open("path/to/watchdog.db")
```

### 3. Initialize in main.go

Replace the beginning of `main()` function in `/home/amedeus/Dev/go-watchdog/cmd/watchdog/main.go` with:

```go
package main

import (
	// ... existing imports ...
	"github.com/Ameprizzo/go-watchdog/internal/database"
	"github.com/Ameprizzo/go-watchdog/internal/database/repositories"
)

func main() {
	// Initialize database
	db, err := database.Open("watchdog.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get repositories
	repos := repositories.New(db)

	// Load config (existing code)
	cfg, err := monitor.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Load settings from database if available
	dbSettings, err := repos.Settings.GetAll()
	if err == nil && dbSettings != nil {
		cfg.Settings.CheckInterval = dbSettings.CheckIntervalSeconds
		cfg.Settings.Timeout = dbSettings.TimeoutSeconds
	}

	// Sync sites from config.json to database
	for _, site := range cfg.Sites {
		exists, _ := repos.Sites.Exists(site.Name)
		if !exists {
			dbSite := &database.Site{
				Name:    site.Name,
				URL:     site.URL,
				Enabled: true,
			}
			repos.Sites.Create(dbSite)
		}
	}

	// ... rest of your code ...
}
```

### 4. Verify Installation

Run your application:
```bash
go run cmd/watchdog/main.go
```

You should see:
```
✅ Database connection established
✅ Site created: Google (ID: 1)
✅ Site created: GitHub (ID: 2)
...
🐕 Watchdog started. Checking 5 sites every 20s...
```

## Database File

After first run, you'll have a new `watchdog.db` file:

```bash
$ ls -lh watchdog.db
-rw-r--r--  1 user  group  20K Jan  8 10:30 watchdog.db
```

## Viewing Database Contents

### Using SQLite CLI
```bash
sqlite3 watchdog.db

# View sites
sqlite> SELECT * FROM sites;

# View settings
sqlite> SELECT * FROM app_settings;

# Check tables
sqlite> .tables
```

### Using a GUI Tool
- [DB Browser for SQLite](https://sqlitebrowser.org/) - Free & simple
- [DBeaver Community](https://dbeaver.io/) - Full-featured

## Troubleshooting

### "cannot find package"
```bash
go mod tidy
go mod download
```

### "database is locked"
This happens with concurrent writes. The implementation handles this, but if you see it:
1. Close all database connections
2. Delete `watchdog.db-wal` and `watchdog.db-shm` files
3. Restart the application

### "table already exists"
This is expected on subsequent runs. The migrations check `IF NOT EXISTS` so they're safe to run repeatedly.

## Next Phase

When Phase 2 is ready, the database will:
1. Record every uptime check
2. Calculate daily statistics
3. Track downtime incidents
4. Store notification history
5. Maintain audit logs

The foundation is now ready!
