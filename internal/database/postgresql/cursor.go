package postgresql

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

type Cursor struct {
	secretKey []byte
}

func NewCursor(secretKey []byte) *Cursor {

	return &Cursor{secretKey: secretKey}
}

func (c *Cursor) Encrypt(id int) (string, error) {
	block, err := aes.NewCipher(c.secretKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext := make([]byte, 4)
	binary.BigEndian.PutUint32(plaintext, uint32(id))

	// Generate a random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	// Encrypt with authentication
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

func (c *Cursor) Decrypt(encodedCursor string) (int, error) {
	data, err := base64.URLEncoding.DecodeString(encodedCursor)
	if err != nil {
		return 0, err
	}

	block, err := aes.NewCipher(c.secretKey)
	if err != nil {
		return 0, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return 0, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, fmt.Errorf("decryption failed: %w", err)
	}

	if len(plaintext) != 4 {
		return 0, fmt.Errorf("invalid plaintext length")
	}

	return int(binary.BigEndian.Uint32(plaintext)), nil
}
