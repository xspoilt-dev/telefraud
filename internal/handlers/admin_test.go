package handlers

import (
	"bytes"
	"testing"
)

func TestNamedBytesReader(t *testing.T) {
	data := []byte("test content")
	nbr := namedBytesReader{
		Reader: bytes.NewReader(data),
		name:   "sample.csv",
	}

	if nbr.Name() != "sample.csv" {
		t.Errorf("Name() = %q, want 'sample.csv'", nbr.Name())
	}

	buf := make([]byte, len(data))
	n, err := nbr.Read(buf)
	if err != nil || n != len(data) || string(buf) != "test content" {
		t.Errorf("Read failed: n=%d err=%v buf=%s", n, err, string(buf))
	}
}
