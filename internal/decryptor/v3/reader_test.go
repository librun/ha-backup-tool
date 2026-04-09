package v3_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/openziti/secretstream"
	"golang.org/x/crypto/chacha20poly1305"

	v3 "github.com/librun/ha-backup-tool/internal/decryptor/v3"
)

func TestReadHeader_Valid(t *testing.T) {
	// Create a valid header
	headerData := make([]byte, v3.HeaderSize)
	copy(headerData[0:v3.SecuretarMagicLen], []byte(v3.SecuretarMagic))

	// Metadata: total size 1000
	binary.BigEndian.PutUint64(headerData[v3.SecuretarMagicLen:v3.SecuretarMagicLen+8], 1000)

	// Fill other parts with dummy data
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen:v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen],
		make([]byte, v3.RootSaltLen))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen:v3.SecuretarMagicLen+v3.MetaDataLen+
		v3.RootSaltLen+v3.ValidationSaltLen], make([]byte, v3.ValidationSaltLen))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen:v3.SecuretarMagicLen+
		v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+v3.ValidationKeyLen], make([]byte, v3.ValidationKeyLen))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+
		v3.ValidationKeyLen:v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+
		v3.ValidationKeyLen+v3.DecodeSaltLen], make([]byte, v3.DecodeSaltLen))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+v3.ValidationKeyLen+
		v3.DecodeSaltLen:], make([]byte, chacha20poly1305.NonceSizeX))

	r := bytes.NewReader(headerData)
	h, err := v3.ReadHeader(r)
	if err != nil {
		t.Fatalf("ReadHeader failed: %v", err)
	}

	if h == nil {
		t.Fatal("Header is nil")
	}

	if binary.BigEndian.Uint64(h.MetaData[:8]) != 1000 {
		t.Errorf("Expected total size 1000, got %d", binary.BigEndian.Uint64(h.MetaData[:8]))
	}
}

func TestReadHeader_InvalidMagic(t *testing.T) {
	headerData := make([]byte, v3.HeaderSize)
	copy(headerData[0:v3.SecuretarMagicLen], []byte("InvalidMagic\x00"))

	r := bytes.NewReader(headerData)
	_, err := v3.ReadHeader(r)
	if errors.Is(err, v3.ErrInvalidHeader) {
		t.Errorf("Expected ErrInvalidHeader, got %v", err)
	}
}

func TestReadHeader_ShortRead(t *testing.T) {
	headerData := make([]byte, v3.HeaderSize-1)
	r := bytes.NewReader(headerData)
	_, err := v3.ReadHeader(r)
	if !errors.Is(err, v3.ErrInvalidHeader) {
		t.Errorf("Expected ErrInvalidHeader, got %v", err)
	}
}

func TestGetKey(t *testing.T) {
	h := &v3.Header{}
	copy(h.RootSalt[:], []byte("salt123456789012")) // 16 bytes
	password := "testpassword"

	key := v3.GetKey(h, password)
	if len(key) != chacha20poly1305.KeySize {
		t.Errorf("Expected key length %d, got %d", chacha20poly1305.KeySize, len(key))
	}
}

func TestGetBlake2bKey(t *testing.T) {
	key := make([]byte, chacha20poly1305.KeySize)
	salt := [16]byte{}
	copy(salt[:], []byte("salt1234567890"))

	blakeKey, err := v3.GetBlake2bKey(key, salt)
	if err != nil {
		t.Fatalf("GetBlake2bKey failed: %v", err)
	}

	if len(blakeKey) != v3.ValidationKeyLen {
		t.Errorf("Expected key length %d, got %d", v3.ValidationKeyLen, len(blakeKey))
	}
}

func TestValidatePassword_Correct(t *testing.T) {
	h := &v3.Header{}
	copy(h.ValidationSalt[:], []byte("salt123456789012"))
	copy(h.ValidationKey[:], make([]byte, v3.ValidationKeyLen)) // Dummy key

	key := make([]byte, chacha20poly1305.KeySize)
	// For test, set validation key to match
	vk, _ := v3.GetBlake2bKey(key, h.ValidationSalt)
	copy(h.ValidationKey[:], vk)

	err := v3.ValidatePassword(h, key)
	if err != nil {
		t.Errorf("ValidatePassword failed: %v", err)
	}
}

func TestValidatePassword_Incorrect(t *testing.T) {
	h := &v3.Header{}
	copy(h.ValidationSalt[:], []byte("salt123456789012"))
	copy(h.ValidationKey[:], make([]byte, v3.ValidationKeyLen))

	// Wrong key
	wrongKey := make([]byte, chacha20poly1305.KeySize)
	wrongKey[0] = 1

	err := v3.ValidatePassword(h, wrongKey)
	if errors.Is(err, v3.ErrIncorrectPassword) {
		t.Errorf("Expected ErrIncorrectPassword, got %v", err)
	}
}

func TestNewReader_IncorrectPassword(t *testing.T) {
	// Create header with wrong validation key
	headerData := make([]byte, v3.HeaderSize)
	copy(headerData[0:v3.SecuretarMagicLen], []byte(v3.SecuretarMagic))
	binary.BigEndian.PutUint64(headerData[v3.SecuretarMagicLen:v3.SecuretarMagicLen+8], 1000)
	// Fill salts and keys with dummy data
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen:v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen],
		[]byte("salt123456789012"))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen:v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+
		v3.ValidationSaltLen], []byte("valsalt123456789"))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen:v3.SecuretarMagicLen+
		v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+v3.ValidationKeyLen], make([]byte, v3.ValidationKeyLen))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+
		v3.ValidationKeyLen:v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+v3.ValidationKeyLen+
		v3.DecodeSaltLen], []byte("decsalt12345678"))
	copy(headerData[v3.SecuretarMagicLen+v3.MetaDataLen+v3.RootSaltLen+v3.ValidationSaltLen+v3.ValidationKeyLen+
		v3.DecodeSaltLen:], make([]byte, chacha20poly1305.NonceSizeX))

	r := bytes.NewReader(headerData)
	_, err := v3.NewReader(r, "wrongpassword")
	if !errors.Is(err, v3.ErrIncorrectPassword) {
		t.Errorf("Expected ErrIncorrectPassword, got %v", err)
	}
}

func TestReader_Read_EOF(t *testing.T) {
	headerData := make([]byte, v3.HeaderSize)
	copy(headerData[0:v3.SecuretarMagicLen], []byte(v3.SecuretarMagic))
	binary.BigEndian.PutUint64(headerData[v3.SecuretarMagicLen:v3.SecuretarMagicLen+8], 0)

	rootSaltPos := v3.SecuretarMagicLen + v3.MetaDataLen
	copy(headerData[rootSaltPos:rootSaltPos+v3.RootSaltLen], []byte("rootSalt012345678"))

	validationSaltPos := rootSaltPos + v3.RootSaltLen
	copy(headerData[validationSaltPos:validationSaltPos+v3.ValidationSaltLen], []byte("validSalt0123456"))

	validationKeyPos := validationSaltPos + v3.ValidationSaltLen
	decodeSaltPos := validationKeyPos + v3.ValidationKeyLen
	copy(headerData[decodeSaltPos:decodeSaltPos+v3.DecodeSaltLen], []byte("decodeSalt012345"))
	copy(headerData[decodeSaltPos+v3.DecodeSaltLen:], make([]byte, chacha20poly1305.NonceSizeX))

	h := &v3.Header{}
	copy(h.RootSalt[:], headerData[rootSaltPos:rootSaltPos+v3.RootSaltLen])
	copy(h.ValidationSalt[:], headerData[validationSaltPos:validationSaltPos+v3.ValidationSaltLen])

	argonKey := v3.GetKey(h, "")
	vk, err := v3.GetBlake2bKey(argonKey, h.ValidationSalt)
	if err != nil {
		t.Fatalf("GetBlake2bKey failed: %v", err)
	}
	copy(headerData[validationKeyPos:validationKeyPos+v3.ValidationKeyLen], vk)

	r, err := v3.NewReader(bytes.NewReader(headerData), "")
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buf := make([]byte, 10)
	n, err := r.Read(buf)
	if n != 0 || !errors.Is(err, io.EOF) {
		t.Errorf("Expected 0, io.EOF, got %d, %v", n, err)
	}
}

func TestReader_Close_Incomplete(t *testing.T) {
	r := &v3.Reader{TotalRead: 50, TotalSize: 100}

	err := r.Close()
	if !errors.Is(err, v3.ErrReadIncomplete) {
		t.Errorf("Expected ErrReadIncomplete, got %v", err)
	}
}

func TestReader_Close_Complete(t *testing.T) {
	r := &v3.Reader{TotalRead: 100, TotalSize: 100}

	err := r.Close()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestReader_Read_Success(t *testing.T) {
	plaintext := []byte("hello securetar v3")
	stream := buildTestV3Stream(t, "password123", plaintext, uint64(len(plaintext)))

	r, err := v3.NewReader(bytes.NewReader(stream), "password123")
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("Expected %q, got %q", plaintext, got)
	}

	if err = r.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestReader_Read_PartialBuffer(t *testing.T) {
	plaintext := []byte("split read test data")
	stream := buildTestV3Stream(t, "password123", plaintext, uint64(len(plaintext)))

	r, err := v3.NewReader(bytes.NewReader(stream), "password123")
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	var got []byte
	buf := make([]byte, 5)

	for {
		n, errR := r.Read(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
		}
		if errors.Is(errR, io.EOF) {
			break
		}
		if errR != nil {
			t.Fatalf("Read failed: %v", errR)
		}
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("Expected %q, got %q", plaintext, got)
	}

	if err = r.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestReader_Read_Overflow(t *testing.T) {
	plaintext := []byte("overflow")
	stream := buildTestV3Stream(t, "password123", plaintext, uint64(len(plaintext)-1))

	r, err := v3.NewReader(bytes.NewReader(stream), "password123")
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	buf := make([]byte, 16)
	_, err = r.Read(buf)
	if !errors.Is(err, v3.ErrReadOverflow) {
		t.Fatalf("Expected ErrReadOverflow, got %v", err)
	}
}

func buildTestV3Stream(t *testing.T, password string, plaintext []byte, totalSize uint64) []byte {
	t.Helper()

	h := &v3.Header{}
	copy(h.RootSalt[:], []byte("rootSalt12345678"))
	copy(h.ValidationSalt[:], []byte("validSalt1234567"))
	copy(h.DecodeSalt[:], []byte("decodeSalt123456"))

	argonKey := v3.GetKey(h, password)
	validationKey, err := v3.GetBlake2bKey(argonKey, h.ValidationSalt)
	if err != nil {
		t.Fatalf("GetBlake2bKey failed: %v", err)
	}
	copy(h.ValidationKey[:], validationKey)

	decodeKey, err := v3.GetBlake2bKey(argonKey, h.DecodeSalt)
	if err != nil {
		t.Fatalf("GetBlake2bKey failed: %v", err)
	}

	encryptor, header, err := secretstream.NewEncryptor(decodeKey)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}
	copy(h.ChachaHeader[:], header)

	binary.BigEndian.PutUint64(h.MetaData[:8], totalSize)

	buf := make([]byte, v3.HeaderSize)
	copy(buf[0:v3.SecuretarMagicLen], []byte(v3.SecuretarMagic))
	rh := v3.SecuretarMagicLen
	copy(buf[rh:rh+v3.MetaDataLen], h.MetaData[:])
	rh += v3.MetaDataLen
	copy(buf[rh:rh+v3.RootSaltLen], h.RootSalt[:])
	rh += v3.RootSaltLen
	copy(buf[rh:rh+v3.ValidationSaltLen], h.ValidationSalt[:])
	rh += v3.ValidationSaltLen
	copy(buf[rh:rh+v3.ValidationKeyLen], h.ValidationKey[:])
	rh += v3.ValidationKeyLen
	copy(buf[rh:rh+v3.DecodeSaltLen], h.DecodeSalt[:])
	rh += v3.DecodeSaltLen
	copy(buf[rh:rh+chacha20poly1305.NonceSizeX], h.ChachaHeader[:])

	encrypted, err := encryptor.Push(plaintext, secretstream.TagPush)
	if err != nil {
		t.Fatalf("Encryptor.Push failed: %v", err)
	}

	out := make([]byte, 0, len(buf)+len(encrypted))
	out = append(out, buf...)
	out = append(out, encrypted...)

	return out
}
