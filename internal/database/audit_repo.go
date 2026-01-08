package database

import (
	"log"
	"time"
)

// AuditLogRepository handles audit log operations
type AuditLogRepository struct {
	db *Database
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *Database) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create inserts a new audit log entry
func (r *AuditLogRepository) Create(audit *AuditLog) error {
	result, err := r.db.Exec(`
		INSERT INTO audit_logs (action, entity_type, entity_id, old_value, new_value, user_id, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, audit.Action, audit.EntityType, audit.EntityID, audit.OldValue, audit.NewValue,
		audit.UserID, audit.Timestamp)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	audit.ID = int(id)
	log.Printf("✅ Audit log created: %s on %s (ID: %d)", audit.Action, audit.EntityType, audit.EntityID)
	return nil
}

// GetByID retrieves an audit log entry by ID
func (r *AuditLogRepository) GetByID(id int) (*AuditLog, error) {
	var audit AuditLog
	err := r.db.QueryRow(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs WHERE id = ?
	`, id).Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID, &audit.OldValue,
		&audit.NewValue, &audit.UserID, &audit.Timestamp)

	if err != nil {
		return nil, err
	}

	return &audit, nil
}

// GetByEntityID retrieves all audit logs for an entity
func (r *AuditLogRepository) GetByEntityID(entityType string, entityID int, limit int) ([]AuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, entityType, entityID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID,
			&audit.OldValue, &audit.NewValue, &audit.UserID, &audit.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, audit)
	}

	return logs, rows.Err()
}

// GetByAction retrieves all audit logs by action type
func (r *AuditLogRepository) GetByAction(action string, limit int) ([]AuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs
		WHERE action = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, action, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID,
			&audit.OldValue, &audit.NewValue, &audit.UserID, &audit.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, audit)
	}

	return logs, rows.Err()
}

// GetByDateRange retrieves audit logs within a date range
func (r *AuditLogRepository) GetByDateRange(startTime, endTime time.Time) ([]AuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs
		WHERE timestamp BETWEEN ? AND ?
		ORDER BY timestamp DESC
	`, startTime, endTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID,
			&audit.OldValue, &audit.NewValue, &audit.UserID, &audit.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, audit)
	}

	return logs, rows.Err()
}

// GetByUserID retrieves all audit logs for a specific user
func (r *AuditLogRepository) GetByUserID(userID string, limit int) ([]AuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID,
			&audit.OldValue, &audit.NewValue, &audit.UserID, &audit.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, audit)
	}

	return logs, rows.Err()
}

// Count returns the total number of audit logs
func (r *AuditLogRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&count)
	return count, err
}

// CountByEntity returns the count of audit logs for an entity
func (r *AuditLogRepository) CountByEntity(entityType string, entityID int) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM audit_logs
		WHERE entity_type = ? AND entity_id = ?
	`, entityType, entityID).Scan(&count)
	return count, err
}

// DeleteOlderThan deletes audit logs older than specified duration
func (r *AuditLogRepository) DeleteOlderThan(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result, err := r.db.Exec(`
		DELETE FROM audit_logs WHERE timestamp < ?
	`, cutoffTime)

	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("✅ Deleted %d old audit logs", rowsAffected)
	}

	return rowsAffected, nil
}

// GetRecent retrieves the most recent audit logs
func (r *AuditLogRepository) GetRecent(limit int) ([]AuditLog, error) {
	rows, err := r.db.Query(`
		SELECT id, action, entity_type, entity_id, old_value, new_value, user_id, timestamp
		FROM audit_logs
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.Action, &audit.EntityType, &audit.EntityID,
			&audit.OldValue, &audit.NewValue, &audit.UserID, &audit.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, audit)
	}

	return logs, rows.Err()
}

// GetActivitySummary returns a summary of actions taken
func (r *AuditLogRepository) GetActivitySummary(days int) (map[string]int, error) {
	startTime := time.Now().AddDate(0, 0, -days)

	rows, err := r.db.Query(`
		SELECT action, COUNT(*) as count
		FROM audit_logs
		WHERE timestamp >= ?
		GROUP BY action
		ORDER BY count DESC
	`, startTime)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[string]int)
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		summary[action] = count
	}

	return summary, rows.Err()
}
