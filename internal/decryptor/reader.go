package decryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

const (
	SecuretarMagic = "SecureTar\x02\x00\x00\x00\x00\x00\x00"
)

var (
	// ErrNotEnoughBytes is returned when reading unwanted number of bytes from data.
	ErrNotEnoughBytes = errors.New("acs: not enough read bytes")
	// ErrTooShort is returned when reading AES CBC data that is too short (less than blocksize).
	ErrTooShort = errors.New("acs: ciphertext too short")
	// ErrModulo is returned when reading AES CBC data that is not a multiple of the block size.
	ErrModulo = errors.New("acs: ciphertext is not a multiple of the block size")
)

// A Reader is an io.Reader that can be read to retrieve decrypted data from a AES CBC crypted file.
type Reader struct {
	beginning bool
	key       []byte
	mu        sync.Mutex
	r         io.Reader
	iv        []byte
	block     cipher.Block
	size      uint64
	mode      cipher.BlockMode
}

// NewAesCbcReader returns an AES-CBC reader.
func NewReader(r io.Reader, passwd string) (*Reader, error) {
	key, err := PasswordToKey(passwd)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &Reader{
		beginning: true,
		key:       key,
		r:         r,
		block:     block,
		iv:        make([]byte, block.BlockSize()),
	}, err
}

// Read implements io.Reader interface,
func (r *Reader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.beginning {
		if n, err := r.readInfoBytes(); err != nil {
			return n, err
		}
	}

	n, err := r.r.Read(p)
	if err != nil && errors.Is(err, io.EOF) {
		return 0, err
	}

	if n == 0 {
		// EOF
		return 0, nil
	}

	run := p[:n]
	if len(run) < r.block.BlockSize() {
		return n, ErrTooShort
	}
	if len(run)%r.block.BlockSize() != 0 {
		return n, ErrModulo
	}

	r.mode.CryptBlocks(run, run)

	return n, err
}

func (r *Reader) readInfoBytes() (int, error) {
	n, err := io.ReadFull(r.r, r.iv)
	if err != nil {
		return 0, err
	}
	if n != len(r.iv) {
		return 0, ErrNotEnoughBytes
	}
	r.beginning = false

	// Securetar added uncrypt info in first 16 bytes
	if string(r.iv) == SecuretarMagic {
		rs := make([]byte, r.block.BlockSize())

		n, err = io.ReadFull(r.r, rs)
		if err != nil {
			return 0, err
		}
		if n != len(rs) {
			return 0, ErrNotEnoughBytes
		}

		r.size = binary.BigEndian.Uint64(rs[:8])

		n, err = io.ReadFull(r.r, r.iv)
		if err != nil {
			return 0, err
		}
		if n != len(r.iv) {
			return 0, ErrNotEnoughBytes
		}
	}

	r.iv, err = GenerateIv(r.key, r.iv)
	if err != nil {
		return 0, err
	}

	r.mode = cipher.NewCBCDecrypter(r.block, r.iv)

	return n, err
}

// GenerateIv - Generate initialization vector.
func GenerateIv(key, salt []byte) ([]byte, error) {
	var b []byte

	b = append(b, key...)
	b = append(b, salt...)

	for range 100 {
		h := sha256.New()
		if _, err := h.Write(b); err != nil {
			return nil, err
		}

		b = h.Sum(nil)
	}

	return b[:aes.BlockSize], nil
}

// PasswordToKey - Convert password/key to encryption key.
func PasswordToKey(password string) ([]byte, error) {
	b := []byte(password)

	for range 100 {
		h := sha256.New()
		if _, err := h.Write(b); err != nil {
			return nil, err
		}

		b = h.Sum(nil)
	}

	return b[:aes.BlockSize], nil
}
