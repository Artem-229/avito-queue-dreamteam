package middlewares

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func Verify(data, signature, secret string) bool {
	expected := Sign(data, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}
