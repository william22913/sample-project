package text

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"strings"
)

func GenerateMessageDigestCompact(
	message []byte,
) (
	string,
	error,
) {
	if string(message) == "" {
		return "", nil
	}

	buffer := new(bytes.Buffer)

	if err := json.Compact(buffer, message); err != nil {
		return "", err
	}

	return CheckSumWithSha256(buffer.Bytes()), nil
}

func GenerateMessageDigest(
	message string,
) string {
	message = strings.Replace(message, "\r", "", -1)
	return CheckSumWithSha256([]byte(message))
}

func GenerateSignature(
	httpMethod string,
	relativeURL string,
	accessToken string,
	messageDigest string,
	timestamp string,
	key string,
) string {
	message := httpMethod + ":" + relativeURL + ":" + accessToken + ":" + messageDigest + ":" + timestamp
	return ChecksumWithHMACSHA(sha512.New, []byte(message), key)
}

func ValidateSignature(
	httpMethod string,
	relativeURL string,
	accessToken string,
	messageDigest string,
	timestamp string,
	key string,
	signature string,
) bool {
	message := httpMethod + ":" + relativeURL + ":" + accessToken + ":" + messageDigest + ":" + timestamp
	return signature == ChecksumWithHMACSHA(sha512.New, []byte(message), key)
}

func SignMessageWith256RSAPrivate(
	privateKey *rsa.PrivateKey,
	message []byte,
) (
	string,
	error,
) {
	hashed := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed[:])

	if err != nil {
		return "", err
	}

	return Base64encoder(signature), nil
}

func CompareMessageWith256RSA(
	publicKey *rsa.PublicKey,
	message []byte,
	signature string,
) error {
	hashed := sha256.Sum256(message)
	signatureBytes, err := Base64decoder(signature)

	if err != nil {
		return err
	}

	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signatureBytes)
}

func ReadPrivateRSAKey(
	data []byte,
) (
	*rsa.PrivateKey,
	error,
) {
	block, _ := pem.Decode(data)

	if block == nil {
		return nil, errors.New("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)

	if err != nil {
		return nil, err
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)

	if !ok {
		return nil, errors.New("parsed private key is not an RSA private key")
	}

	return rsaPrivateKey, nil
}

func ReadPublicRSAKey(
	data []byte,
) (
	*rsa.PublicKey,
	error,
) {
	block, _ := pem.Decode(data)

	if block == nil {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)

	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)

	if !ok {
		return nil, errors.New("parsed private key is not an RSA private key")
	}

	return rsaPublicKey, nil
}
