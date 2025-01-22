package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"strings"
)

const defaultRootFolder = "/home/happypotter/dfs"

type PathTransformFunc func(string) PathKey

type StoreOpts struct {
	//Root is the folder path to the root on the disk containing all the
	//folder structure
	Root string
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
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolder
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(key string) bool {
	PathKey := s.PathTransformFunc(key)
	_, err := os.Stat(s.Root + string(os.PathSeparator) + PathKey.FilePath())
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {
		log.Printf("delted [%s] from disk", s.Root+string(os.PathSeparator)+pathKey.FilePath())
	}()
	return os.RemoveAll(s.Root + string(os.PathSeparator) + pathKey.FilePath())
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

func (s *Store) Write(key string, r io.Reader) error {
	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)
	return os.Open(s.Root + string(os.PathSeparator) + pathkey.FilePath())
}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(s.Root+string(os.PathSeparator)+pathKey.pathName, os.ModePerm); err != nil {
		return err
	}

	FilePath := pathKey.FilePath()

	f, err := os.Create(s.Root + string(os.PathSeparator) + FilePath)

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
