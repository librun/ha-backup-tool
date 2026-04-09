package decryptor

import (
	"errors"
	"strings"

	"github.com/librun/ha-backup-tool/internal/entity"
	"golang.org/x/mod/semver"
)

type Decryptor int

const (
	DecryptorSecureTarAuto Decryptor = iota
	DecryptorSecureTarV1
	DecryptorSecureTarV2
	DecryptorSecureTarV3
)

const (
	DecryptorSecureTarAutoString = "auto"
	DecryptorSecureTarV1String   = "v1"
	DecryptorSecureTarV2String   = "v2"
	DecryptorSecureTarV3String   = "v3"
	DecryptorUnknownString       = "unknown"
)

const (
	DecryptorSecureTarV3From = "2026.3.4"
	DecryptorAES128          = "aes128"
)

var (
	ErrDecryptorUnknown        = errors.New("decryptor not support")
	ErrDecryptorV1NotSupported = errors.New("SecureTar v1 not support")
)

func (d Decryptor) String() string {
	switch d {
	case DecryptorSecureTarAuto:
		return DecryptorSecureTarAutoString
	case DecryptorSecureTarV1:
		return DecryptorSecureTarV1String
	case DecryptorSecureTarV2:
		return DecryptorSecureTarV2String
	case DecryptorSecureTarV3:
		return DecryptorSecureTarV3String
	default:
		return DecryptorUnknownString
	}
}

func ParseFromString(s string) (Decryptor, error) {
	switch strings.ToLower(s) {
	case "":
		return DecryptorSecureTarAuto, nil
	case DecryptorSecureTarV1String:
		return DecryptorSecureTarV1, ErrDecryptorV1NotSupported
	case DecryptorSecureTarV2String:
		return DecryptorSecureTarV2, nil
	case DecryptorSecureTarV3String:
		return DecryptorSecureTarV3, nil
	}

	return 0, ErrDecryptorUnknown
}

func ParseFromBackupJSON(e *entity.HomeAssistantBackup, d Decryptor) (Decryptor, error) {
	if !strings.EqualFold(e.Crypto, DecryptorAES128) {
		return 0, ErrDecryptorUnknown
	}

	if d != DecryptorSecureTarAuto {
		return d, nil
	}

	c := semver.Compare("v"+DecryptorSecureTarV3From, "v"+e.Homeassistant.Version)
	if c > 0 {
		return DecryptorSecureTarV2, nil
	}

	return DecryptorSecureTarV3, nil
}
