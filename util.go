package miservice

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func generateRandomStr(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func md5Hash(value string) string {
	h := md5.New()
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

func generateNonce() (string, error) {
	// Generate 8 random bytes
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	// Get the current time in minutes
	currentTime := time.Now().Unix() / 60
	timeBytes := make([]byte, 4)
	timeBytes[0] = byte(currentTime >> 24)
	timeBytes[1] = byte(currentTime >> 16)
	timeBytes[2] = byte(currentTime >> 8)
	timeBytes[3] = byte(currentTime)

	// Concatenate random bytes and time bytes
	nonceBytes := append(randomBytes, timeBytes...)

	// Encode to base64
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)

	return nonce, nil
}

func signNonce(ssecurity string, nonce string) (string, error) {
	// Decode the base64 strings
	ssecurityDecoded, err := base64.StdEncoding.DecodeString(ssecurity)
	if err != nil {
		return "", err
	}

	nonceDecoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", err
	}

	// Create a new SHA-256 hash
	hash := sha256.New()

	// Write the decoded ssecurity and nonce to the hash
	hash.Write(ssecurityDecoded)
	hash.Write(nonceDecoded)

	// Compute the SHA-256 checksum
	hashSum := hash.Sum(nil)

	// Encode the checksum to base64
	signedNonce := base64.StdEncoding.EncodeToString(hashSum)

	return signedNonce, nil
}

type Data interface {
	Parse(token *Token, cookies Cookies) (map[string]any, error)
}

type DataFunc func(token *Token, cookies Cookies) (map[string]any, error)

func (f DataFunc) Parse(token *Token, cookies Cookies) (map[string]any, error) {
	return f(token, cookies)
}

type DataMap map[string]any

func (d DataMap) Parse(_ *Token, _ Cookies) (map[string]any, error) {
	return d, nil
}

type Cookies map[string]string

func (c Cookies) Write(req *http.Request) {
	if c == nil {
		return
	}
	var buf bytes.Buffer
	for k, v := range c {
		buf.WriteString(fmt.Sprintf("%s=%s; ", k, v))
	}
	req.Header.Set("Cookie", buf.String())
}
