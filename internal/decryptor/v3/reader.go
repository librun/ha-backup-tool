package v3

import (
	"bytes"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/minio/blake2b-simd"
	"github.com/openziti/secretstream"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	SecuretarMagic = "SecureTar\x03"
	blake2bPerson  = "SecureTarv3"

	SecuretarMagicLen = 16
	MetaDataLen       = 16
	RootSaltLen       = 16
	ValidationSaltLen = 16
	ValidationKeyLen  = 32
	DecodeSaltLen     = 16

	argonTimeLimit = 8
	argonMemLimit  = 16 * 1024 // 16 MB
	argonThreads   = 1

	HeaderSize = SecuretarMagicLen + MetaDataLen + RootSaltLen + ValidationSaltLen +
		ValidationKeyLen + DecodeSaltLen + chacha20poly1305.NonceSizeX

	secretStreamChunkDataSize = 1024 * 1024 // V3_SECRETSTREAM_CHUNK_SIZE
	secretStreamChunkSize     = secretStreamChunkDataSize + secretstream.StreamABytes
)

var (
	ErrInvalidHeader     = errors.New("invalid header")
	ErrIncorrectPassword = errors.New("incorrect password")
	ErrReadOverflow      = errors.New("read overflow")
	ErrReadIncomplete    = errors.New("incomplete read")
	ErrFailedToGetBuffer = errors.New("failed to get buffer from pool")

	//nolint:gochecknoglobals // buffer for reading raw encrypted data
	bufferPool = sync.Pool{
		New: func() any {
			b := make([]byte, secretStreamChunkSize)

			return &b
		},
	}
)

type Reader struct {
	reader        io.Reader
	decryptor     secretstream.Decryptor
	decryptedData []byte
	Offset        int
	TotalRead     uint64
	TotalSize     uint64
}

type Header struct {
	MetaData       [MetaDataLen]byte
	RootSalt       [RootSaltLen]byte
	ValidationSalt [ValidationSaltLen]byte
	ValidationKey  [ValidationKeyLen]byte
	DecodeSalt     [DecodeSaltLen]byte
	ChachaHeader   [chacha20poly1305.NonceSizeX]byte
}

func NewReader(r io.Reader, password string) (*Reader, error) {
	h, err := ReadHeader(r)
	if err != nil {
		return nil, err
	}

	argonKey := GetKey(h, password)

	if err = ValidatePassword(h, argonKey); err != nil {
		return nil, err
	}

	dk, err := GetBlake2bKey(argonKey, h.DecodeSalt)
	if err != nil {
		return nil, err
	}

	d, err := secretstream.NewDecryptor(dk, h.ChachaHeader[:])
	if err != nil {
		return nil, err
	}

	return &Reader{
		reader:    r,
		decryptor: d,
		TotalSize: binary.BigEndian.Uint64(h.MetaData[:8]),
	}, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	if len(r.decryptedData)-r.Offset >= len(p) {
		n := copy(p, r.decryptedData[r.Offset:r.Offset+len(p)])
		r.Offset += n

		return n, nil
	}

	n := 0
	if r.Offset < len(r.decryptedData) {
		n = copy(p, r.decryptedData[r.Offset:])
		r.Offset = len(r.decryptedData)
	}

	if err := r.GetNextChunk(); err != nil {
		if !errors.Is(err, io.EOF) || n == 0 {
			return 0, err
		}
	}

	if len(r.decryptedData) == 0 {
		if n > 0 {
			return n, nil
		}

		return 0, io.EOF
	}

	m := copy(p[n:], r.decryptedData)
	n += m
	r.Offset = m

	return n, nil
}

func (r *Reader) GetNextChunk() error {
	b, ok := bufferPool.Get().(*[]byte)
	if !ok {
		return ErrFailedToGetBuffer
	}
	defer bufferPool.Put(b)

	r.decryptedData = r.decryptedData[:0]
	r.Offset = 0

	n, err := io.ReadFull(r.reader, *b)
	if err != nil {
		switch {
		// ignore unexpected EOF if we read some data, because it can be last chunk
		case errors.Is(err, io.ErrUnexpectedEOF):
		// ignore if we read some data, because it can be last chunk
		case errors.Is(err, io.EOF) && n > 0:
		default:
			return err
		}
	}

	if n == 0 {
		return io.EOF
	}

	decrypted, _, err := r.decryptor.Pull((*b)[:n])
	if err != nil {
		return err
	}

	r.decryptedData = append(r.decryptedData[:0], decrypted...)
	if uint64(len(r.decryptedData))+r.TotalRead > r.TotalSize {
		return ErrReadOverflow
	}

	r.TotalRead += uint64(len(r.decryptedData))

	return nil
}

// Close - close reader and check if all data was read.
func (r *Reader) Close() error {
	if r.TotalSize != r.TotalRead {
		return ErrReadIncomplete
	}

	return nil
}

func ReadHeader(r io.Reader) (*Header, error) {
	b := make([]byte, HeaderSize)

	n, err := io.ReadFull(r, b)
	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
			return nil, ErrInvalidHeader
		}

		return nil, err
	}

	if n != HeaderSize || !bytes.HasPrefix(b, []byte(SecuretarMagic)) {
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
	rh := SecuretarMagicLen

	copy(h.MetaData[:], b[rh:rh+MetaDataLen]) // metadata
	rh += MetaDataLen

	copy(h.RootSalt[:], b[rh:rh+RootSaltLen]) // rootSalt
	rh += RootSaltLen

	copy(h.ValidationSalt[:], b[rh:rh+ValidationSaltLen]) // Validation key salt
	rh += ValidationSaltLen

	copy(h.ValidationKey[:], b[rh:rh+ValidationKeyLen]) // Validation derived key
	rh += ValidationKeyLen

	copy(h.DecodeSalt[:], b[rh:rh+DecodeSaltLen]) // Secret stream key salt
	rh += DecodeSaltLen

	copy(h.ChachaHeader[:], b[rh:rh+chacha20poly1305.NonceSizeX]) // Cipher header

	return &h, nil
}

func GetKey(h *Header, password string) []byte {
	return argon2.IDKey(
		[]byte(password),
		h.RootSalt[:],
		argonTimeLimit,
		argonMemLimit,
		argonThreads,
		chacha20poly1305.KeySize,
	)
}

func GetBlake2bKey(key []byte, salt [16]byte) ([]byte, error) {
	// golang not support salt and personal in standard lib issue https://github.com/golang/go/issues/32447
	// after issue is closed replace archive package "github.com/minio/blake2b-simd" to "golang.org/x/crypto/blake2b"
	h, err := blake2b.New(&blake2b.Config{
		Size:   ValidationKeyLen,
		Key:    key,
		Salt:   salt[:],
		Person: []byte(blake2bPerson),
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
