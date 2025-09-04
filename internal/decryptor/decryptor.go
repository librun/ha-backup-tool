package decryptor

import (
	"io"

	"github.com/librun/ha-backup-tool/internal/decryptor/aes128"
)

func New(r io.Reader, t Decryptor, passwd string) (io.Reader, error) {
	switch t {
	case DecryptorAES128:
		return aes128.NewReader(r, passwd)
	default:
		return nil, ErrDecryptorUnknown
	}
}
