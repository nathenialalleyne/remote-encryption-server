package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func main() {

}

func encrypt(plaintext string, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil{
		return "", err
	}

	cipherText :=make([]byte, aes.BlockSize+len(plaintext))
	iv := cipherText[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], []byte(plaintext))

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func generateAESKey(length int) ([]byte, error){
	key := make([]byte, length)

	_, err := io.ReadFull(rand.Reader, key)
	if err != nil{
		return nil, fmt.Errorf("Failed to generate key: %v", err)
	}

	return key, nil
}