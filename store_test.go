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
	s := newStore()

	defer teardown(t, s)
	key := "mybestpicturessdf"
	data := []byte("some jpg")

	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if ok := s.Has(key); !ok {
		t.Errorf("expected to have key %s", key)
	}
	if ok := s.Has("mybestpicture2"); ok {
		t.Errorf("expected to not have key %s", "mybestpicture2")
	}
	_, r, err := s.Read(key)
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

	s := newStore()

	key := "mybestpictures"

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
