package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strings"
)

type PathTransformFunc func(string) PathKey

type StoreOpts struct {
	PathTransformFunc
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{pathName: key, fileName: key}
}

type PathKey struct {
	pathName string
	fileName string
}

func (p PathKey) FilePath() string {
	return p.pathName + string(os.PathSeparator) + p.fileName
}

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashString) / blocksize

	path := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		path[i] = hashString[from:to]
	}
	return PathKey{
		pathName: strings.Join(path, string(os.PathSeparator)),
		fileName: hashString,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	PathKey := s.PathTransformFunc(key)
	_, err := os.Stat(PathKey.FilePath())
	if err == os.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {
		log.Printf("delted [%s] from disk", pathKey.fileName)
	}()
	return os.RemoveAll(pathKey.FilePath())
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)
	return os.Open(pathkey.FilePath())
}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(pathKey.pathName, os.ModePerm); err != nil {
		return err
	}

	FilePath := pathKey.FilePath()

	f, err := os.Create(FilePath)

	if err != nil {
		return err
	}
	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	log.Printf("written (%d) bytes to disk: %s", n, FilePath)

	return nil
}
