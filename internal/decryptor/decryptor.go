package decryptor

import (
	"io"

	v2 "github.com/librun/ha-backup-tool/internal/decryptor/v2"
	v3 "github.com/librun/ha-backup-tool/internal/decryptor/v3"
)

func New(r io.Reader, t Decryptor, passwd string) (io.ReadCloser, error) {
	switch t {
	case DecryptorSecureTarAuto:
	case DecryptorSecureTarV1:
		return nil, ErrDecryptorV1NotSupported
	case DecryptorSecureTarV2:
		return v2.NewReader(r, passwd)
	case DecryptorSecureTarV3:
		return v3.NewReader(r, passwd)
	}

	return nil, ErrDecryptorUnknown
}
