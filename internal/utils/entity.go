package utils

import "time"

type (
	HomeAssistantBackup struct {
		Slug              string    `json:"slug"`
		Version           int       `json:"version"`
		Name              string    `json:"name"`
		Date              time.Time `json:"date"`
		Type              string    `json:"type"`
		SupervisorVersion string    `json:"supervisor_version"`
		Crypto            string    `json:"crypto"`
		Protected         bool      `json:"protected"`
		Compressed        bool      `json:"compressed"`
		Homeassistant     struct {
			Version         string  `json:"version"`
			ExcludeDatabase bool    `json:"exclude_database"`
			Size            float64 `json:"size"`
		} `json:"homeassistant"`
		Extra struct {
			InstanceID                  string    `json:"instance_id"`
			WithAutomaticSettings       bool      `json:"with_automatic_settings"`
			SupervisorBackupRequestDate time.Time `json:"supervisor.backup_request_date"`
		} `json:"extra"`
		Repositories []string `json:"repositories"`
	}
)
