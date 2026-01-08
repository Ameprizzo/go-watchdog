package database

import (
	"log"
	"time"
)

// NotificationRepository handles notification log operations
type NotificationRepository struct {
	db *Database
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *Database) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create inserts a new notification log
func (r *NotificationRepository) Create(notif *NotificationLog) error {
	result, err := r.db.Exec(`
		INSERT INTO notification_logs (notification_id, site_id, type, message, severity, sent_at, status, retry_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, notif.NotificationID, notif.SiteID, notif.Type, notif.Message, notif.Severity,
		notif.SentAt, notif.Status, notif.RetryCount)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	notif.ID = int(id)
	log.Printf("✅ Notification logged: %s (Type: %s, Status: %s)", notif.NotificationID, notif.Type, notif.Status)
	return nil
}

// Update updates a notification log
func (r *NotificationRepository) Update(notif *NotificationLog) error {
	_, err := r.db.Exec(`
		UPDATE notification_logs SET status = ?, retry_count = ? WHERE id = ?
	`, notif.Status, notif.RetryCount, notif.ID)

	return err
}

// GetByID retrieves a notification log by ID
func (r *NotificationRepository) GetByID(id int) (*NotificationLog, error) {
	var notif NotificationLog
	err := r.db.QueryRow(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs WHERE id = ?
	`, id).Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type, &notif.Message,
		&notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount)

	if err != nil {
		return nil, err
	}

	return &notif, nil
}

// GetBySiteID retrieves all notification logs for a site
func (r *NotificationRepository) GetBySiteID(siteID int, limit int) ([]NotificationLog, error) {
	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE site_id = ?
		ORDER BY sent_at DESC
		LIMIT ?
	`, siteID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}

// GetPendingNotifications retrieves notifications that need to be sent (status = 'pending')
func (r *NotificationRepository) GetPendingNotifications(limit int) ([]NotificationLog, error) {
	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE status = 'pending'
		ORDER BY sent_at ASC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}

// GetFailedNotifications retrieves notifications that failed to send
func (r *NotificationRepository) GetFailedNotifications(limit int) ([]NotificationLog, error) {
	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE status = 'failed'
		ORDER BY sent_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}

// GetByDateRange retrieves notifications within a date range
func (r *NotificationRepository) GetByDateRange(siteID int, startTime, endTime time.Time) ([]NotificationLog, error) {
	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE site_id = ? AND sent_at BETWEEN ? AND ?
		ORDER BY sent_at DESC
	`, siteID, startTime, endTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}

// Count returns the total number of notification logs
func (r *NotificationRepository) Count(siteID int) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM notification_logs WHERE site_id = ?", siteID).Scan(&count)
	return count, err
}

// GetBySeverity retrieves notifications by severity level
func (r *NotificationRepository) GetBySeverity(siteID int, severity string, limit int) ([]NotificationLog, error) {
	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE site_id = ? AND severity = ?
		ORDER BY sent_at DESC
		LIMIT ?
	`, siteID, severity, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}

// DeleteOlderThan deletes notification logs older than specified duration
func (r *NotificationRepository) DeleteOlderThan(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result, err := r.db.Exec(`
		DELETE FROM notification_logs WHERE sent_at < ?
	`, cutoffTime)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("✅ Deleted %d old notification logs", rowsAffected)
	}

	return rowsAffected, nil
}

// MarkAllAsSent marks all pending notifications as sent
func (r *NotificationRepository) MarkAllAsSent() error {
	_, err := r.db.Exec(`
		UPDATE notification_logs SET status = 'sent' WHERE status = 'pending'
	`)
	return err
}

// GetRecentByType retrieves recent notifications by type
func (r *NotificationRepository) GetRecentByType(siteID int, notifType string, hours int) ([]NotificationLog, error) {
	startTime := time.Now().Add(-time.Hour * time.Duration(hours))

	rows, err := r.db.Query(`
		SELECT id, notification_id, site_id, type, message, severity, sent_at, status, retry_count
		FROM notification_logs
		WHERE site_id = ? AND type = ? AND sent_at >= ?
		ORDER BY sent_at DESC
	`, siteID, notifType, startTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []NotificationLog
	for rows.Next() {
		var notif NotificationLog
		if err := rows.Scan(&notif.ID, &notif.NotificationID, &notif.SiteID, &notif.Type,
			&notif.Message, &notif.Severity, &notif.SentAt, &notif.Status, &notif.RetryCount); err != nil {
			return nil, err
		}
		notifications = append(notifications, notif)
	}

	return notifications, rows.Err()
}
