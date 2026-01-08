package database

import (
	"log"
	"time"
)

// SiteRepository handles site operations
type SiteRepository struct {
	db *Database
}

// NewSiteRepository creates a new site repository
func NewSiteRepository(db *Database) *SiteRepository {
	return &SiteRepository{db: db}
}

// GetAll retrieves all sites
func (r *SiteRepository) GetAll() ([]Site, error) {
	rows, err := r.db.Query("SELECT id, name, url, enabled, current_status, last_checked, created_at, updated_at FROM sites ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Site
	for rows.Next() {
		var site Site
		if err := rows.Scan(&site.ID, &site.Name, &site.URL, &site.Enabled, &site.CurrentStatus, &site.LastChecked, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}

	return sites, rows.Err()
}

// GetByID retrieves a site by ID
func (r *SiteRepository) GetByID(id int) (*Site, error) {
	var site Site
	err := r.db.QueryRow("SELECT id, name, url, enabled, current_status, last_checked, created_at, updated_at FROM sites WHERE id = ?", id).
		Scan(&site.ID, &site.Name, &site.URL, &site.Enabled, &site.CurrentStatus, &site.LastChecked, &site.CreatedAt, &site.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &site, nil
}

// GetByName retrieves a site by name
func (r *SiteRepository) GetByName(name string) (*Site, error) {
	var site Site
	err := r.db.QueryRow("SELECT id, name, url, enabled, current_status, last_checked, created_at, updated_at FROM sites WHERE name = ?", name).
		Scan(&site.ID, &site.Name, &site.URL, &site.Enabled, &site.CurrentStatus, &site.LastChecked, &site.CreatedAt, &site.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &site, nil
}

// Create inserts a new site
func (r *SiteRepository) Create(site *Site) error {
	now := time.Now()
	result, err := r.db.Exec(
		"INSERT INTO sites (name, url, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		site.Name, site.URL, site.Enabled, now, now,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	site.ID = int(id)
	site.CreatedAt = now
	site.UpdatedAt = now

	log.Printf("✅ Site created: %s (ID: %d)", site.Name, site.ID)
	return nil
}

// Update updates an existing site
func (r *SiteRepository) Update(site *Site) error {
	site.UpdatedAt = time.Now()

	_, err := r.db.Exec(
		"UPDATE sites SET name = ?, url = ?, enabled = ?, updated_at = ? WHERE id = ?",
		site.Name, site.URL, site.Enabled, site.UpdatedAt, site.ID,
	)

	if err != nil {
		return err
	}

	log.Printf("✅ Site updated: %s (ID: %d)", site.Name, site.ID)
	return nil
}

// Delete removes a site by ID
func (r *SiteRepository) Delete(id int) error {
	site, err := r.GetByID(id)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("DELETE FROM sites WHERE id = ?", id)
	if err != nil {
		return err
	}

	log.Printf("✅ Site deleted: %s (ID: %d)", site.Name, id)
	return nil
}

// UpdateStatus updates the current status of a site
func (r *SiteRepository) UpdateStatus(id int, status string, lastChecked time.Time) error {
	_, err := r.db.Exec(
		"UPDATE sites SET current_status = ?, last_checked = ?, updated_at = ? WHERE id = ?",
		status, lastChecked, time.Now(), id,
	)
	return err
}

// UpdateStatusByName updates status by site name
func (r *SiteRepository) UpdateStatusByName(name string, status string, lastChecked time.Time) error {
	site, err := r.GetByName(name)
	if err != nil {
		return err
	}
	return r.UpdateStatus(site.ID, status, lastChecked)
}

// Count returns the total number of sites
func (r *SiteRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM sites").Scan(&count)
	return count, err
}

// Exists checks if a site exists by name
func (r *SiteRepository) Exists(name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM sites WHERE name = ?)", name).Scan(&exists)
	return exists, err
}
