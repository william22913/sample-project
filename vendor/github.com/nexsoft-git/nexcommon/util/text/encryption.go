package text

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func NewEncryptionHelper(
	key string,
) (
	EncryptionHelper,
	error,
) {

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	result := encryptionHelper{
		block: block,
	}

	return &result, nil
}

type EncryptionHelper interface {
	EncryptMessage(string) (string, error)
	DecryptMessage(string) (string, error)
	SetForURLEncryption()
}

type encryptionHelper struct {
	block  cipher.Block
	forURL bool
}

func (e *encryptionHelper) SetForURLEncryption() {
	e.forURL = true
}

func (e encryptionHelper) EncryptMessage(
	plaintext string,
) (
	string,
	error,
) {

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(e.block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(plaintext))

	if e.forURL {
		return base64.URLEncoding.EncodeToString(ciphertext), nil
	} else {
		return base64.StdEncoding.EncodeToString(ciphertext), nil
	}
}

func (e encryptionHelper) DecryptMessage(
	encodedMessage string,
) (
	string,
	error,
) {

	var ciphertext []byte
	var err error

	if e.forURL {
		ciphertext, err = base64.URLEncoding.DecodeString(encodedMessage)
		if err != nil {
			return "", err
		}
	} else {
		ciphertext, err = base64.StdEncoding.DecodeString(encodedMessage)
		if err != nil {
			return "", err
		}
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("Ciphertext is too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(e.block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}
