package postgresql

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

// Cursor 페이지네이션 커서(마지막 id)를 AES-GCM 으로 암호화한다. 클라이언트가 id 를 추측·조작하지 못한다.
type Cursor struct {
	aead cipher.AEAD
}

// NewCursor secretKey 는 16, 24, 32바이트 중 하나(AES-128/192/256)
func NewCursor(secretKey []byte) (*Cursor, error) {
	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return nil, fmt.Errorf("cursor secret: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cursor{aead: aead}, nil
}

func (c *Cursor) Encrypt(id int64) (string, error) {
	plaintext := binary.BigEndian.AppendUint64(nil, uint64(id))

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(c.aead.Seal(nonce, nonce, plaintext, nil)), nil
}

func (c *Cursor) Decrypt(encoded string) (int64, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, errors.New("invalid cursor")
	}

	nonceSize := c.aead.NonceSize()
	if len(data) < nonceSize {
		return 0, errors.New("invalid cursor")
	}

	plaintext, err := c.aead.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil || len(plaintext) != 8 {
		return 0, errors.New("invalid cursor")
	}

	return int64(binary.BigEndian.Uint64(plaintext)), nil
}
