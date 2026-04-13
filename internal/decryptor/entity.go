package decryptor

import (
	"errors"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/librun/ha-backup-tool/internal/entity"
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
	v3FromSupervisor = ">= 2026.3.1"
	v3FromCore       = ">= 2026.3.0"
	cryptoAES128     = "aes128"
)

var (
	ErrDecryptorUnknown        = errors.New("decryptor not support")
	ErrGetVersion              = errors.New("version in config not valid")
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
	if d != DecryptorSecureTarAuto {
		return d, nil
	}

	vsc, err := version.NewConstraint(v3FromSupervisor)
	if err != nil {
		return 0, ErrGetVersion
	}
	vs, err := version.NewVersion(e.SupervisorVersion)
	if err != nil {
		return 0, ErrGetVersion
	}

	vcc, err := version.NewConstraint(v3FromCore)
	if err != nil {
		return 0, ErrGetVersion
	}
	vc, err := version.NewVersion(e.Homeassistant.Version)
	if err != nil {
		return 0, ErrGetVersion
	}

	if !vsc.Check(vs) || !vcc.Check(vc) {
		if !strings.EqualFold(e.Crypto, cryptoAES128) {
			return 0, ErrDecryptorUnknown
		}

		return DecryptorSecureTarV2, nil
	}

	return DecryptorSecureTarV3, nil
}
