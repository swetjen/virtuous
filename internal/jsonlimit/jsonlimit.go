package jsonlimit

import (
	"errors"
	"io"
	"net/http"
)

const DefaultMaxBytes int64 = 1 << 20

var ErrBodyTooLarge = errors.New("json request body too large")

func MaxBytesReader(w http.ResponseWriter, r *http.Request, maxBytes int64) io.Reader {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	return http.MaxBytesReader(w, r.Body, maxBytes)
}

func LimitReader(r *http.Request, maxBytes int64) (io.Reader, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if r.ContentLength > maxBytes {
		return nil, ErrBodyTooLarge
	}
	return &reader{r: r.Body, remaining: maxBytes}, nil
}

func IsBodyTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.Is(err, ErrBodyTooLarge) || errors.As(err, &maxBytesErr)
}

type reader struct {
	r         io.Reader
	remaining int64
}

func (r *reader) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, ErrBodyTooLarge
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.r.Read(p)
	r.remaining -= int64(n)
	return n, err
}
