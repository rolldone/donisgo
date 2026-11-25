// Package secrets encrypts and decrypts small blobs (typically credentials or
// tokens) with the application master key.
//
//	enc, err := secrets.EncryptBlob([]byte("sensitive"))
//	plain, err := secrets.DecryptBlob(enc)
//
// EncryptBlob returns a JSON envelope; DecryptBlob passes plain (non-envelope)
// input through unchanged, which makes both safe to use against legacy data.
// EnsureEncryptedBlob encrypts only values that are still plaintext.
//
// The master key comes from APP_KEY (preferred) or S2S_KEY, base64/hex encoded.
package secrets
