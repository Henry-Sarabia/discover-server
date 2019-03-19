package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
)

// concatBuf concatenates two strings inside a byte buffer
func concatBuf(a, b string) bytes.Buffer {
	var buf bytes.Buffer
	buf.WriteString(a)
	buf.WriteString(b)
	return buf
}

// hash returns the HMAC hash of the provided slice of bytes using SHA-256.
func hash(b []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, hashKey)
	_, err := mac.Write(b)
	if err != nil {
		return nil, err
	}
	h := mac.Sum(nil)
	return h, nil
}
