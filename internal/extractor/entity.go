package extractor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	decryptor "github.com/librun/ha-backup-tool/internal/decryptor"
	"github.com/librun/ha-backup-tool/internal/entity"
	"github.com/librun/ha-backup-tool/internal/logger"
)

type BackupConfig struct {
	e         *entity.HomeAssistantBackup
	decryptor decryptor.Decryptor `json:"-"`
}

func NewBackupConfig(compressed bool) *BackupConfig {
	return &BackupConfig{e: &entity.HomeAssistantBackup{Compressed: compressed}}
}

func BackupConfigUnmarshalJSON(fpath string) (*BackupConfig, error) {
	var bc BackupConfig
	fo, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = fo.Close(); err != nil {
			logger.Fatalf("File: %s Error close file: %s", fpath, err)
		}
	}()

	b, err := io.ReadAll(fo)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, &bc.e); err != nil {
		return nil, err
	}

	return &bc, nil
}

func (b *BackupConfig) InitAndValidate() error {
	var vs bool

	var errD error
	if b.decryptor, errD = decryptor.ParseFromBackupJSON(b.e, b.decryptor); errD != nil {
		if errors.Is(errD, decryptor.ErrDecryptorUnknown) {
			return fmt.Errorf("crypto type %s not support", b.e.Crypto) //nolint:err113 // Dynamic error
		}

		return errD
	}

	for _, s := range backupJSONVersionSupport {
		if s == b.e.Version {
			vs = true
		}
	}

	if !vs {
		return fmt.Errorf("version backup %d not support", b.e.Version) //nolint:err113 // Dynamic error
	}

	return nil
}

func (b *BackupConfig) GetDecryptor() decryptor.Decryptor {
	return b.decryptor
}

func (b *BackupConfig) SetProtected(v bool) {
	b.e.Protected = v
}

func (b *BackupConfig) IsProtected() bool {
	return b.e.Protected
}

func (b *BackupConfig) IsCompressed() bool {
	return b.e.Compressed
}
