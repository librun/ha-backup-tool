package decryptor

import (
	"testing"
)

func TestGenerateIv(t *testing.T) {
	key := []byte("\xf1\x98\xef\xe53H[H\x1f\xad\x84\xe3\x08\xee\xb4\x92")
	salt := []byte("\xcb\x1e\xaf\x15\x02\xb0\xe2\x88\xa8=\xb0\x10\xd5\x1c\xbf\x07")
	r := "m\x1c\xe4\xc4\x96|m\r!\nM\x16\x02\xf9\x8d\xbc"

	v, _ := generateIv(key, salt)

	if r != string(v) {
		t.Errorf("Expected %s got %s", r, v)
	}
}
