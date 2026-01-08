# How to Run Go-Watchdog

## Correct Way (Recommended)

Use the package path, not the individual file:

```bash
# Start the watchdog application
go run ./cmd/watchdog

# Or run the compiled binary
./watchdog
```

## Why Not `go run cmd/watchdog/main.go`?

When you run `go run cmd/watchdog/main.go`, Go only compiles that single file, ignoring other `.go` files in the same package like `api.go`. This causes undefined function errors.

The correct approach is to use `go run ./cmd/watchdog` which:
- ✅ Compiles all `.go` files in the `cmd/watchdog` package
- ✅ Includes `main.go` and `api.go` together
- ✅ Properly resolves function calls between files

## Expected Output

When started correctly, you'll see:

```
2026/01/08 23:11:42 ✅ Database connection established
2026/01/08 23:11:42 ✅ Maintenance service started (Aggregation: 01:00, Cleanup: 03:00, Retention: 90 days)
🐕 Watchdog started. Checking 5 sites every 20s...

--- Check started at 23:11:42 ---
🌐 Web dashboard running at http://localhost:8080
✅ My Odoo              | Latency: 59ms | Status: 200
✅ Odoo VFD             | Latency: 688ms | Status: 200
✅ GitHub               | Latency: 769ms | Status: 200
✅ Google               | Latency: 1.053s | Status: 200
✅ My Portfolio         | Latency: 1.536s | Status: 200
```

## Access Points

Once running, access the dashboard at:

- **Dashboard:** http://localhost:8080
- **Analytics:** http://localhost:8080/uptime-detail?site_id=1
- **Reports:** http://localhost:8080/reports
- **Manage Sites:** http://localhost:8080/manage

## Building the Binary

To create a compiled binary for distribution:

```bash
go build ./cmd/watchdog
./watchdog
```

This creates a standalone `watchdog` binary that doesn't require Go to be installed.
