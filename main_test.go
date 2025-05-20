package main

import (
	"testing"
)

func TestPasswordToKey(t *testing.T) {
	passwd := "XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
	r := "\xf1\x98\xef\xe53H[H\x1f\xad\x84\xe3\x08\xee\xb4\x92"
	// f198efe533485b481fad84e308eeb492v
	// [241, 152, 239, 229, 51, 72, 91, 72, 31, 173, 132, 227, 8, 238, 180, 146]

	v, _ := passwordToKey(passwd)
	if r != string(v) {
		t.Errorf("Expected %s got %s", r, v)
	}
}
