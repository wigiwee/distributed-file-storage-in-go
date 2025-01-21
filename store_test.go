package main

import (
	"bytes"
	"testing"
)

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
	}
	s := NewStore(opts)

	data := bytes.NewReader([]byte("some jpg"))
	if err := s.writeStream("mySpecalPicture", data); err != nil {
		t.Error(err)
	}
}
