package main

import (
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
		log.Printf("deleted [%s] from disk", s.Root+string(os.PathSeparator)+pathKey.FilePath())
	}()
	return os.RemoveAll(s.Root + string(os.PathSeparator) + pathKey.FilePath())
}

// todo : instread of copying directly to the reader we first copy this into
// a bffer, maybe just return the file from readStream
func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) WriteDecrypt(enc []byte, key string, r io.Reader) (int64, error) {

	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}

	n, err := copyDecrypt(enc, r, f)
	return int64(n), err
}

func (s *Store) openFileForWriting(key string) (*os.File, error) {

	pathKey := s.PathTransformFunc(key)

	if err := os.MkdirAll(s.Root+string(os.PathSeparator)+pathKey.pathName, os.ModePerm); err != nil {
		return nil, err
	}

	return os.Create(s.Root + string(os.PathSeparator) + pathKey.FilePath())

}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}

	return io.Copy(f, r)

}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)

	file, err := os.Open(s.Root + string(os.PathSeparator) + pathkey.FilePath())
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil
}
