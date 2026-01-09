package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Ameprizzo/go-watchdog/internal/database"
)

// setupSyncAndBackupRoutes configures sync and backup endpoints
func setupSyncAndBackupRoutes(repos *database.Repositories) {
	// setupAPIRoutes is called from main, syncService and backupService are global

	// ============ SYNC ENDPOINTS ============

	// POST /api/sync/config-to-db - Sync config.json to database
	http.HandleFunc("/api/sync/config-to-db", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if globalSyncService == nil {
			http.Error(w, "Sync service not initialized", http.StatusInternalServerError)
			return
		}

		result, err := globalSyncService.SyncConfigToDB()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":  err.Error(),
				"result": result,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// GET /api/sync/status - Get sync status
	http.HandleFunc("/api/sync/status", func(w http.ResponseWriter, r *http.Request) {
		if globalSyncService == nil {
			http.Error(w, "Sync service not initialized", http.StatusInternalServerError)
			return
		}

		status := globalSyncService.GetSyncStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})

	// POST /api/sync/config-export - Export DB config
	http.HandleFunc("/api/sync/config-export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if globalSyncService == nil {
			http.Error(w, "Sync service not initialized", http.StatusInternalServerError)
			return
		}

		sites, err := globalSyncService.SyncDBToConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sites": sites,
			"count": len(sites),
		})
	})

	// ============ BACKUP ENDPOINTS ============

	// POST /api/backup/create - Create a new backup
	http.HandleFunc("/api/backup/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if globalBackupService == nil {
			http.Error(w, "Backup service not initialized", http.StatusInternalServerError)
			return
		}

		result, err := globalBackupService.CreateBackup()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(result)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// GET /api/backup/list - List all available backups
	http.HandleFunc("/api/backup/list", func(w http.ResponseWriter, r *http.Request) {
		if globalBackupService == nil {
			http.Error(w, "Backup service not initialized", http.StatusInternalServerError)
			return
		}

		backups, err := globalBackupService.ListBackups()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backups": backups,
			"count":   len(backups),
		})
	})

	// POST /api/backup/restore - Restore from backup
	http.HandleFunc("/api/backup/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if globalBackupService == nil {
			http.Error(w, "Backup service not initialized", http.StatusInternalServerError)
			return
		}

		backupFile := r.FormValue("backup_file")
		if backupFile == "" {
			http.Error(w, "backup_file parameter required", http.StatusBadRequest)
			return
		}

		result, err := globalBackupService.RestoreBackup(backupFile)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(result)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// POST /api/backup/verify - Verify backup integrity
	http.HandleFunc("/api/backup/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if globalBackupService == nil {
			http.Error(w, "Backup service not initialized", http.StatusInternalServerError)
			return
		}

		backupFile := r.FormValue("backup_file")
		if backupFile == "" {
			http.Error(w, "backup_file parameter required", http.StatusBadRequest)
			return
		}

		valid, err := globalBackupService.VerifyBackupIntegrity(backupFile)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backup_file": backupFile,
			"valid":       valid,
			"error": func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		})
	})

	// GET /api/backup/size - Get total backup storage size
	http.HandleFunc("/api/backup/size", func(w http.ResponseWriter, r *http.Request) {
		if globalBackupService == nil {
			http.Error(w, "Backup service not initialized", http.StatusInternalServerError)
			return
		}

		size, err := globalBackupService.GetBackupSize()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_backup_size_bytes": size,
			"total_backup_size_mb":    float64(size) / 1024 / 1024,
		})
	})

	log.Printf("✅ Sync & Backup API routes registered (9 endpoints)")
}
