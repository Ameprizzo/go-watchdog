package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	conn *sql.DB
	mu   sync.RWMutex
}

var instance *Database
var once sync.Once

// Open initializes and returns a singleton database connection
func Open(dbPath string) (*Database, error) {
	var err error
	once.Do(func() {
		conn, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return
		}

		// Enable foreign keys
		if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return
		}

		// Configure connection pool
		conn.SetMaxOpenConns(25)
		conn.SetMaxIdleConns(5)

		// Test connection
		if err := conn.Ping(); err != nil {
			return
		}

		instance = &Database{conn: conn}

		// Run migrations
		if err := instance.runMigrations(); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		log.Println("✅ Database connection established")
	})

	return instance, err
}

// GetDB returns the singleton database instance
func GetDB() *Database {
	return instance
}

// Close closes the database connection
func (db *Database) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// runMigrations executes all database migrations
func (db *Database) runMigrations() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	migrations := []string{
		createSitesTable,
		createAppSettingsTable,
		createUptimeRecordsTable,
		createUptimeSummaryTable,
		createDowntimeIncidentsTable,
		createNotificationLogsTable,
		createAuditLogsTable,
		createIndexes,
	}

	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Initialize default settings if not exists
	if err := db.initializeDefaultSettings(); err != nil {
		return err
	}

	return nil
}

// initializeDefaultSettings creates default settings if they don't exist
func (db *Database) initializeDefaultSettings() error {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM app_settings").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err := db.conn.Exec(`
			INSERT INTO app_settings (
				check_interval_seconds, timeout_seconds, retention_days, 
				sla_target_percentage, enable_history, theme, timezone
			) VALUES (20, 10, 90, 99.9, 1, 'dark', 'UTC')
		`)
		return err
	}

	return nil
}

// Exec executes a query that doesn't return rows
func (db *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.conn.Exec(query, args...)
}

// Query executes a query that returns rows
func (db *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.conn.Query(query, args...)
}

// QueryRow executes a query that returns a single row
func (db *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.conn.QueryRow(query, args...)
}

// BeginTx starts a transaction
func (db *Database) BeginTx() (*sql.Tx, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	return db.conn.Begin()
}

// Migration SQL statements
const (
	createSitesTable = `
		CREATE TABLE IF NOT EXISTS sites (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			current_status TEXT DEFAULT 'unknown',
			last_checked DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	createAppSettingsTable = `
		CREATE TABLE IF NOT EXISTS app_settings (
			id INTEGER PRIMARY KEY,
			check_interval_seconds INTEGER DEFAULT 20,
			timeout_seconds INTEGER DEFAULT 10,
			retention_days INTEGER DEFAULT 90,
			sla_target_percentage REAL DEFAULT 99.9,
			enable_history BOOLEAN DEFAULT 1,
			theme TEXT DEFAULT 'dark',
			timezone TEXT DEFAULT 'UTC',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	createUptimeRecordsTable = `
		CREATE TABLE IF NOT EXISTS uptime_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			status_code INTEGER,
			is_up BOOLEAN NOT NULL,
			latency_ms INTEGER,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE
		)
	`

	createUptimeSummaryTable = `
		CREATE TABLE IF NOT EXISTS uptime_summary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			date DATE NOT NULL,
			total_checks INTEGER DEFAULT 0,
			successful_checks INTEGER DEFAULT 0,
			uptime_percentage REAL DEFAULT 0.0,
			avg_latency_ms REAL DEFAULT 0.0,
			min_latency_ms INTEGER,
			max_latency_ms INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(site_id, date),
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE
		)
	`

	createDowntimeIncidentsTable = `
		CREATE TABLE IF NOT EXISTS downtime_incidents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_seconds INTEGER DEFAULT 0,
			incident_count INTEGER DEFAULT 1,
			root_cause TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE
		)
	`

	createNotificationLogsTable = `
		CREATE TABLE IF NOT EXISTS notification_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id TEXT UNIQUE NOT NULL,
			site_id INTEGER,
			type TEXT NOT NULL,
			message TEXT,
			severity TEXT,
			sent_at DATETIME NOT NULL,
			status TEXT DEFAULT 'pending',
			retry_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(site_id) REFERENCES sites(id) ON DELETE CASCADE
		)
	`

	createAuditLogsTable = `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id INTEGER,
			old_value TEXT,
			new_value TEXT,
			user_id TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`

	createIndexes = `
		CREATE INDEX IF NOT EXISTS idx_uptime_site_time 
		ON uptime_records(site_id, timestamp);
		
		CREATE INDEX IF NOT EXISTS idx_uptime_summary_site_date 
		ON uptime_summary(site_id, date);
		
		CREATE INDEX IF NOT EXISTS idx_incidents_site 
		ON downtime_incidents(site_id);
		
		CREATE INDEX IF NOT EXISTS idx_notification_logs_site 
		ON notification_logs(site_id);
		
		CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp 
		ON audit_logs(timestamp);
	`
)
