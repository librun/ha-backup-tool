package decryptor

import (
	"errors"
	"strings"

	"github.com/librun/ha-backup-tool/internal/entity"
	"golang.org/x/mod/semver"
)

type Decryptor int

const (
	DecryptorSecureTarEmpty Decryptor = iota
	DecryptorSecureTarV2
	DecryptorSecureTarV3
)

const (
	DecryptorSecureTarV2String = "v2"
	DecryptorSecureTarV3String = "v3"
	DecryptorUnknownString     = "unknown"
)

const (
	DecryptorSecureTarV3From = "2026.3.4"
	DecryptorAES128          = "aes128"
)

var (
	ErrDecryptorIsEmpty = errors.New("decryptor not set")
	ErrDecryptorUnknown = errors.New("decryptor not support")
)

func (d Decryptor) String() string {
	switch d {
	case DecryptorSecureTarEmpty:
		return ""
	case DecryptorSecureTarV2:
		return DecryptorSecureTarV2String
	case DecryptorSecureTarV3:
		return DecryptorSecureTarV2String
	default:
		return DecryptorUnknownString
	}
}

func ParseFromString(s string) (Decryptor, error) {
	switch strings.ToLower(s) {
	case "":
		return DecryptorSecureTarEmpty, nil
	case DecryptorSecureTarV2String:
		return DecryptorSecureTarV2, nil
	case DecryptorSecureTarV3String:
		return DecryptorSecureTarV3, nil
	}

	return 0, ErrDecryptorUnknown
}

func ParseFromBackupJSON(e *entity.HomeAssistantBackup) (Decryptor, error) {
	if !strings.EqualFold(e.Crypto, DecryptorAES128) {
		return 0, ErrDecryptorUnknown
	}

	c := semver.Compare("v"+DecryptorSecureTarV3From, "v"+e.Homeassistant.Version)
	if c > 0 {
		return DecryptorSecureTarV2, nil
	}

	return DecryptorSecureTarV3, nil
}
