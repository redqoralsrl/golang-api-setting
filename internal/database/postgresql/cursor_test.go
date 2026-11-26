package postgresql

import (
	"strings"
	"testing"
)

func TestCursorRoundTrip(t *testing.T) {
	c, err := NewCursor([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{0, 1, 1 << 40} {
		enc, err := c.Encrypt(id)
		if err != nil {
			t.Fatal(err)
		}
		got, err := c.Decrypt(enc)
		if err != nil || got != id {
			t.Errorf("round trip %d: got %d, err %v", id, got, err)
		}
	}
}

func TestCursorRejectsTampered(t *testing.T) {
	c, _ := NewCursor([]byte(strings.Repeat("k", 32)))
	other, _ := NewCursor([]byte(strings.Repeat("o", 32)))
	enc, _ := other.Encrypt(7)

	for _, s := range []string{enc, "", "!!!", "AAAA"} {
		if _, err := c.Decrypt(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestNewCursorRejectsBadKey(t *testing.T) {
	if _, err := NewCursor([]byte("short")); err == nil {
		t.Fatal("expected error")
	}
}
