package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func newEncryptionKey() []byte {

	keyBuf := make([]byte, 32)

	io.ReadFull(rand.Reader, keyBuf)
	return keyBuf
}

func copyDecrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	//read the iv
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return 0, err
	}
	stream := cipher.NewCTR(block, iv)
	return copyStream(stream, block.BlockSize(), src, dest)

	// var (
	// 	buf    = make([]byte, 1) // 32 * 1024 is size of io reader/writer buffer
	// 	nw     = block.BlockSize()
	// )

	// for {
	// 	n, err := src.Read(buf)
	// 	if n > 0 {
	// 		stream.XORKeyStream(buf, buf[:n])
	// 		nWritten, err := dest.Write(buf)
	// 		if err != nil {
	// 			return 0, err
	// 		}
	// 		nw += nWritten
	// 	}
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		return 0, err
	// 	}
	// }
	// return nw, err
}

func copyEncrypt(key []byte, src io.Reader, dest io.Writer) (int, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return 0, err
	}

	//prepend the iv to the file
	if _, err := dest.Write(iv); err != nil {
		return 0, err
	}

	stream := cipher.NewCTR(block, iv)
	// var (
	// 	buf    = make([]byte, 32*1024) // 32 * 1024 is size of io reader/writer buffer
	// 	nw     = block.BlockSize()
	// )
	return copyStream(stream, block.BlockSize(), src, dest)

}

func copyStream(stream cipher.Stream, blocksize int, src io.Reader, dest io.Writer) (int, error) {

	var (
		buf = make([]byte, 32*1024) // 32 * 1024 is size of io reader/writer buffer
		nw  = blocksize
	)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			nn, err := dest.Write(buf[:n])
			if err != nil {
				return 0, err
			}
			nw = nw + nn
		}
		if err == io.EOF {
			return 0, err
		}
		if err != nil {
			return 0, err
		}
	}
	return nw, nil
}
