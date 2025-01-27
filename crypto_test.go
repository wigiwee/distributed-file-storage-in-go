package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// func TestNewEncryptionKey(t *testing.T) {
// 	for j := 0; j < 3; j++ {
// 		// key := newEncryptionKey()
// 		// fmt.Println(string(key))
// 	}
// }

func TestCopyEncryptDecrypt(t *testing.T) {

	originalText := "Foo not Bar"
	// fmt.Println("Original text :", originalText)
	src := bytes.NewReader([]byte(originalText))

	dest := new(bytes.Buffer)
	key := newEncryptionKey()

	_, err := copyEncrypt(key, src, dest)
	if err != nil {
		t.Error(err.Error())
	}

	fmt.Println(len(originalText))
	fmt.Println(len(dest.String()))
	// fmt.Println("Encrypted text: ", dest.String())
	out := new(bytes.Buffer)

	nw, err := copyDecrypt(key, dest, out)
	if err != nil {
		t.Error(err)
	}

	if nw != 16+len(originalText) {
		t.Fail()
	}

	if strings.EqualFold(originalText, out.String()) {
		t.Errorf("%s encrypted gave %s", originalText, dest.String())
	}

	if strings.EqualFold(originalText, out.String()) {
		t.Errorf("%s encrypted and decrypted gave %s", originalText, out.String())
	}

	// fmt.Println(len(dest.String()))
	// fmt.Println("Decrypted text", out.String())

}
