package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

type envelope struct {
	V      int    `json:"v"`
	Alg    string `json:"alg"`
	IV     string `json:"iv"`
	Cipher string `json:"cipher"`
}

// IsEncryptedBlob reports whether the provided blob looks like an encrypted
// envelope produced by EncryptBlob.
func IsEncryptedBlob(blob string) bool {
	var env envelope
	if err := json.Unmarshal([]byte(blob), &env); err != nil {
		return false
	}
	return env.V > 0 && env.Alg != "" && env.IV != "" && env.Cipher != ""
}

// MasterKey returns the decoded master key used for encrypt/decrypt.
// It prefers APP_KEY and falls back to S2S_KEY for compatibility.
func MasterKey() ([]byte, error) {
	keyB64 := os.Getenv("APP_KEY")
	if keyB64 == "" {
		keyB64 = os.Getenv("S2S_KEY")
	}
	if keyB64 == "" {
		return nil, errors.New("APP_KEY or S2S_KEY is not set")
	}
	if strings.HasPrefix(keyB64, "base64:") {
		keyB64 = strings.TrimPrefix(keyB64, "base64:")
	}
	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("master key must decode to 32 bytes")
	}
	return key, nil
}

// EncryptBlob encrypts plaintext and returns an envelope JSON string.
func EncryptBlob(plaintext []byte) (string, error) {
	key, err := MasterKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	env := envelope{
		V:      1,
		Alg:    "AES-GCM",
		IV:     base64.StdEncoding.EncodeToString(nonce),
		Cipher: base64.StdEncoding.EncodeToString(ciphertext),
	}
	b, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// EnsureEncryptedBlob encrypts plaintext blobs and passes through blobs that
// already look encrypted.
func EnsureEncryptedBlob(blob string) (string, error) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return "", nil
	}
	if IsEncryptedBlob(blob) {
		return blob, nil
	}
	return EncryptBlob([]byte(blob))
}

// DecryptBlob attempts to parse an envelope and decrypt it. If the input
// is plain JSON (not an envelope), it returns the original bytes unchanged.
func DecryptBlob(blob string) ([]byte, error) {
	var env envelope
	if err := json.Unmarshal([]byte(blob), &env); err == nil && env.Cipher != "" && env.IV != "" {
		key, err := MasterKey()
		if err != nil {
			return nil, err
		}
		nonce, err := base64.StdEncoding.DecodeString(env.IV)
		if err != nil {
			return nil, err
		}
		ciphertext, err := base64.StdEncoding.DecodeString(env.Cipher)
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		if len(nonce) != gcm.NonceSize() {
			return nil, errors.New("invalid nonce size")
		}
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return nil, err
		}
		return plaintext, nil
	}
	// Fallback: return original bytes (assume plain JSON)
	return []byte(blob), nil
}
