package v3

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"io"

	"github.com/minio/blake2b-simd"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	securetarMagicLen = 16
	metaDataLen       = 16
	rootSaltLen       = 16
	validationSaltLen = 16
	validationKeyLen  = 32
	decodeSaltLen     = 16

	argnoTimeLimit = 8
	argonMemLimit  = 16 * 1024 // 16 MB
	argonThreads   = 1

	headerSize = securetarMagicLen + metaDataLen + rootSaltLen + validationSaltLen +
		validationKeyLen + decodeSaltLen + chacha20poly1305.NonceSizeX
)

var (
	ErrInvalidHeader     = errors.New("invalid header")
	ErrIncorrectPassword = errors.New("incorrect password")

	blake2bPerson  = [11]byte{'S', 'e', 'c', 'u', 'r', 'e', 'T', 'a', 'r', 'v', '3'}
	securetarMagic = [10]byte{'S', 'e', 'c', 'u', 'r', 'e', 'T', 'a', 'r', '\x03'}
)

type Reader struct {
	r *io.Reader
}

type Header struct {
	MetaData       [metaDataLen]byte
	RootSalt       [rootSaltLen]byte
	ValidationSalt [validationSaltLen]byte
	ValidationKey  [validationKeyLen]byte
	DecodeSalt     [decodeSaltLen]byte
	ChachaHeader   [chacha20poly1305.NonceSizeX]byte
}

func NewReader(r io.Reader, password string) (*Reader, error) {
	h, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	// create key argon2
	argonKey := GetKey(h, password)

	if err := ValidatePassword(h, argonKey); err != nil {
		return nil, err
	}

	dk, err := GetBlake2bKey(argonKey, h.DecodeSalt)
	if err != nil {
		return nil, err
	}

	_ = dk
	// FIXME: create chacha secure stream reader

	return &Reader{r: nil}, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	// FIXME: decode chank
	return 0, nil
}

func ReadHeader(r io.Reader) (*Header, error) {
	// read headers
	b := make([]byte, headerSize)

	n, err := io.ReadFull(r, b)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	if n < headerSize || !bytes.HasPrefix(b, securetarMagic[:]) {
		return nil, ErrInvalidHeader
	}

	h := Header{}

	// # SecureTar v3 header consists of:
	// # 16 bytes file ID: 9 bytes magic + 1 byte version + 6 bytes reserved
	// # 16 bytes file metadata: 8 bytes plaintext size + 8 bytes reserved
	// # 104 bytes cipher initialization:
	// #  - 16 bytes root salt
	// #  - 16 bytes validation salt
	// #  - 32 bytes validation key
	// #  - 16 bytes validation salt
	// #  - 24 bytes cipher header + nonce
	rh := securetarMagicLen

	copy(h.MetaData[:], b[rh:rh+metaDataLen]) // metadata
	rh += metaDataLen

	copy(h.RootSalt[:], b[rh:rh+rootSaltLen]) // rootSalt
	rh += rootSaltLen

	copy(h.ValidationSalt[:], b[rh:rh+validationSaltLen]) // Validation key salt
	rh += validationSaltLen

	copy(h.ValidationKey[:], b[rh:rh+validationKeyLen]) // Validation derived key
	rh += validationKeyLen

	copy(h.DecodeSalt[:], b[rh:rh+decodeSaltLen]) // Secret stream key salt
	rh += decodeSaltLen

	copy(h.ChachaHeader[:], b[rh:rh+chacha20poly1305.NonceSizeX]) // Cipher header

	return &h, nil
}

func GetKey(h *Header, password string) []byte {
	return argon2.IDKey(
		[]byte(password),
		h.RootSalt[:],
		argnoTimeLimit,
		argonMemLimit,
		argonThreads,
		chacha20poly1305.KeySize,
	)
}

func GetBlake2bKey(key []byte, salt [16]byte) ([]byte, error) {
	// golang not support salt and personal in standart lib issue https://github.com/golang/go/issues/32447
	// after close issue replace archive package "github.com/minio/blake2b-simd" to "golang.org/x/crypto/blake2b"
	h, err := blake2b.New(&blake2b.Config{
		Size:   validationKeyLen,
		Key:    key,
		Salt:   salt[:],
		Person: blake2bPerson[:],
	})
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func ValidatePassword(h *Header, key []byte) error {
	vk, err := GetBlake2bKey(key, h.ValidationSalt)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare(vk, h.ValidationKey[:]) != 1 {
		return ErrIncorrectPassword
	}

	return nil
}
