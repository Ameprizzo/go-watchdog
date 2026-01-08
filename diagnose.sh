#!/bin/bash
# Quick diagnostics script for go-watchdog database

echo "🔍 Go-Watchdog Database Diagnostics"
echo "===================================="
echo ""

# Check if database exists
if [ ! -f "watchdog.db" ]; then
    echo "❌ watchdog.db not found!"
    exit 1
fi

echo "✅ Database found: watchdog.db"
echo ""

echo "📊 DATABASE STATISTICS"
echo "---------------------"

# Uptime records count
UPTIME_COUNT=$(sqlite3 watchdog.db "SELECT COUNT(*) FROM uptime_records;" 2>/dev/null)
echo "Uptime Records: $UPTIME_COUNT"

# Incidents count
INCIDENT_COUNT=$(sqlite3 watchdog.db "SELECT COUNT(*) FROM downtime_incidents;" 2>/dev/null)
echo "Downtime Incidents: $INCIDENT_COUNT"

# Summaries count
SUMMARY_COUNT=$(sqlite3 watchdog.db "SELECT COUNT(*) FROM uptime_summary;" 2>/dev/null)
echo "Daily Summaries: $SUMMARY_COUNT"

# Notification logs count
NOTIF_COUNT=$(sqlite3 watchdog.db "SELECT COUNT(*) FROM notification_logs;" 2>/dev/null)
echo "Notification Logs: $NOTIF_COUNT"

# Audit logs count
AUDIT_COUNT=$(sqlite3 watchdog.db "SELECT COUNT(*) FROM audit_logs;" 2>/dev/null)
echo "Audit Logs: $AUDIT_COUNT"

echo ""
echo "📍 SITES IN DATABASE"
echo "--------------------"
sqlite3 watchdog.db "SELECT id, name, current_status FROM sites;" 2>/dev/null | column -t -s'|'

echo ""
echo "📈 RECENT UPTIME RECORDS (Last 5)"
echo "---------------------------------"
sqlite3 watchdog.db "SELECT site_id, timestamp, is_up, latency_ms FROM uptime_records ORDER BY timestamp DESC LIMIT 5;" 2>/dev/null

echo ""
echo "🔴 ACTIVE INCIDENTS (Ongoing)"
echo "-----------------------------"
sqlite3 watchdog.db "SELECT site_id, start_time FROM downtime_incidents WHERE end_time IS NULL;" 2>/dev/null

echo ""
echo "📅 DAILY SUMMARIES (Last 7 days)"
echo "--------------------------------"
sqlite3 watchdog.db "SELECT site_id, date, uptime_percentage FROM uptime_summary ORDER BY date DESC LIMIT 7;" 2>/dev/null

echo ""
echo "🔧 MAINTENANCE SERVICE STATUS"
echo "-----------------------------"
curl -s http://localhost:8080/api/admin/stats 2>/dev/null | jq '.maintenance_status' || echo "API not responding"

echo ""
echo "✅ Diagnostics complete!"
