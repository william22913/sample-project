package text

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"strconv"

	"github.com/OneOfOne/xxhash"
)

func CheckSumWithXXHASH(content []byte) (checksum string) {
	hash := xxhash.Checksum64(content)
	return strconv.Itoa(int(hash))
}

func CheckSumWithMD5(content []byte) (checksum string) {
	hash := md5.New()
	hash.Write(content)
	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes)
}

func CheckSumWithSha256(content []byte) string {
	result := sha256.Sum256(content)
	return hex.EncodeToString(result[:])
}

func CheckSumWithSha512(content []byte) string {
	result := sha512.Sum512(content)
	return hex.EncodeToString(result[:])
}

func ChecksumWithHMACSHA(f func() hash.Hash, content []byte, key string) string {
	h := hmac.New(f, []byte(key))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
