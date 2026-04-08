package v3

import (
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/minio/blake2b-simd"
	"github.com/openziti/secretstream"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	securetarMagic = "SecureTar\x03"
	blake2bPerson  = "SecureTarv3"

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

	secretStreamChunkDataSize = 1024 * 1024 // V3_SECRETSTREAM_CHUNK_SIZE
	secretStreamChunkSize     = secretStreamChunkDataSize + secretstream.StreamABytes
)

var (
	ErrInvalidHeader     = errors.New("invalid header")
	ErrIncorrectPassword = errors.New("incorrect password")
	ErrReadOverflow      = errors.New("read overflow")
	ErrReadIncompleted   = errors.New("incomplete read")
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
	offset        int
	totalRead     uint64
	totalSize     uint64
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

	if err = ValidatePassword(h, argonKey); err != nil {
		return nil, err
	}

	dk, err := GetBlake2bKey(argonKey, h.DecodeSalt)
	if err != nil {
		return nil, err
	}

	_ = dk
	d, err := secretstream.NewDecryptor(dk, h.ChachaHeader[:])
	if err != nil {
		return nil, err
	}

	ts := binary.BigEndian.Uint64(h.MetaData[:8])

	return &Reader{reader: r, decryptor: d, totalSize: ts}, nil
}

func (r *Reader) Read(p []byte) (int, error) {
	// if we have decrypted data in buffer, return it
	if len(r.decryptedData)-r.offset >= len(p) {
		n := copy(p, r.decryptedData[r.offset:r.offset+len(p)])
		r.offset += len(p)

		return n, nil
	}

	// if we have decrypted data in buffer, return it and get next chunk
	var n int
	if len(r.decryptedData)-r.offset > 0 {
		n = copy(p, r.decryptedData[r.offset:])
	}

	if err := r.getNextChunk(); err != nil {
		if !errors.Is(err, io.EOF) {
			return 0, err
		}

		// if we read some data, return it and ignore EOF, because it can be last chunk
		if n == 0 {
			return 0, err
		}
	}

	lr := len(p) - n
	if lr > 0 {
		if lr > len(r.decryptedData) {
			lr = len(r.decryptedData)
		}

		n += copy(p[n:], r.decryptedData[0:lr])
		r.offset = lr
	}

	return n, nil
}

func (r *Reader) getNextChunk() error {
	b, ok := bufferPool.Get().(*[]byte)
	if !ok {
		return ErrFailedToGetBuffer
	}
	defer bufferPool.Put(b)

	clear(r.decryptedData)
	r.decryptedData = r.decryptedData[:0]
	r.offset = 0

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
		// EOF
		return nil
	}

	var t byte
	r.decryptedData, t, err = r.decryptor.Pull((*b)[:n])
	if err != nil {
		return err
	}

	if uint64(len(r.decryptedData))+r.totalRead > r.totalSize {
		return ErrReadOverflow
	}

	r.totalRead += uint64(len(r.decryptedData))

	_ = t

	return nil
}

func (r *Reader) Close() error {
	if r.totalSize != r.totalRead {
		return ErrReadIncompleted
	}

	return nil
}

func ReadHeader(r io.Reader) (*Header, error) {
	// read headers
	b := make([]byte, headerSize)

	n, err := io.ReadFull(r, b)
	if err != nil && errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}

	if n < headerSize || !strings.HasPrefix(string(b), securetarMagic) {
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
