package decryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"errors"
	"io"
	"sync"
)

const (
	SECURETAR_MAGIC = "SecureTar\x02\x00\x00\x00\x00\x00\x00"
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
	mode      cipher.BlockMode
}

// NewAesCbcReader returns an AES-CBC reader.
func NewReader(r io.Reader, key []byte) (*Reader, error) {
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
func (r *Reader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.beginning {
		n, err = io.ReadFull(r.r, r.iv)
		if err != nil {
			n = 0 // Force caller reader (aka gzip.Reader) to catch the error
			return
		}
		if n != len(r.iv) {
			n = 0 // Force caller reader (aka gzip.Reader) to catch the error
			err = ErrNotEnoughBytes
			return
		}
		r.beginning = false

		r.iv, err = generateIv(r.key, r.iv)
		if err != nil {
			n = 0 // Force caller reader (aka gzip.Reader) to catch the error
			return
		}

		r.mode = cipher.NewCBCDecrypter(r.block, r.iv)
	}

	n, err = r.r.Read(p)
	if err != nil && err != io.EOF {
		n = 0 // Force caller reader (aka gzip.Reader) to catch the error
		return
	}

	if n == 0 {
		// EOF
		return
	}

	run := p[:n]
	if len(run) < r.block.BlockSize() {
		err = ErrTooShort
		return
	}
	if len(run)%r.block.BlockSize() != 0 {
		err = ErrModulo
		return
	}

	r.mode.CryptBlocks(run, run)

	return
}

// generateIv - Generate initialization vector.
func generateIv(key, salt []byte) ([]byte, error) {
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
