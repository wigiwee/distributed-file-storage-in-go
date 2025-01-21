package main

import (
	"bytes"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {

	key := "mybestpictures"

	expectedPathname := "7037c/79055/7f0d8/61c53/d3bbd/1fafe/02dc3/699e6"
	expectedOriginalKey := "7037c790557f0d861c53d3bbd1fafe02dc3699e6"

	path := CASPathTransformFunc(key)
	if path.pathName != expectedPathname {
		t.Error(t, "have %s want %s ", path.pathName, expectedPathname)
	}
	if path.fileName != expectedOriginalKey {
		t.Error(t, "have %s want %s ", path.fileName, expectedOriginalKey)
	}
}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "mybestpictures"
	data := []byte("some jpg")

	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Errorf("failed to read from the reader")
	}
	if string(b) != string(data) {
		t.Errorf("want %s want %s", data, b)
	}
}

func TestDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)

	key := "mybestpictures2"

	s.Delete(key)
}
