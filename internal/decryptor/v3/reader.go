package v3

import (
	"errors"
	"io"
	"sync"
)

type Reader struct {
	mu sync.Mutex
	r  io.Reader
}

func NewReader(r io.Reader, passwd string) (*Reader, error) {
	return &Reader{
		r: r,
	}, errors.New("not support v3")
}

func (r *Reader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return 0, nil
}
