package extractor

import (
	"errors"
	"fmt"
	"time"

	decryptor "github.com/librun/ha-backup-tool/internal/decryptor"
)

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

		decryptor decryptor.Decryptor `json:"-"`
	}
)

func (e *HomeAssistantBackup) InitAndValidate() error {
	var vs bool

	var errD error
	if e.decryptor, errD = decryptor.ParseFromString(e.Crypto); errD != nil {
		if errors.Is(errD, decryptor.ErrDecryptorUnknown) {
			return fmt.Errorf("crypto type %s not support", e.Crypto) //nolint:err113 // Dynamic error
		}

		return errD
	}

	for _, s := range backupJSONVersionSupport {
		if s == e.Version {
			vs = true
		}
	}

	if !vs {
		return fmt.Errorf("version backup %d not support", e.Version) //nolint:err113 // Dynamic error
	}

	return nil
}

func (e *HomeAssistantBackup) GetDecryptor() decryptor.Decryptor {
	return e.decryptor
}
