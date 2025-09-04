package decryptor

import (
	"errors"
	"strings"
)

type Decryptor int

const (
	DecryptorAES128 Decryptor = iota
)

const (
	DecryptorAES128String  = "aes128"
	DecryptorUnknownString = "unknown"
)

var (
	ErrDecryptorUnknown = errors.New("decryptor not support")
)

func (d Decryptor) String() string {
	switch d {
	case DecryptorAES128:
		return DecryptorAES128String
	default:
		return DecryptorUnknownString
	}
}

func ParseFromString(s string) (Decryptor, error) {
	switch strings.ToLower(s) {
	case "", DecryptorAES128String:
		return DecryptorAES128, nil
	default:
		return 0, ErrDecryptorUnknown
	}
}
