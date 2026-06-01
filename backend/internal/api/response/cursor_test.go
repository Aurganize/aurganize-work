package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCursorRoundTrip(t *testing.T) {
	c := &Cursor{
		CreatedAt: time.Date(2026, 5, 14, 18, 30, 35, 123456789, time.UTC),
		ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	}

	encodedCursor := c.EncodeCursor()

	if encodedCursor == "" {
		t.Fatalf("encoded cursor empty")
	}

	decoded, err := DecodeCursor(encodedCursor)
	if err != nil {
		t.Fatalf("decode cursor failed : %v", err)
	}

	if !decoded.CreatedAt.Equal(c.CreatedAt) {
		t.Errorf("CreatedAt : got %v; want %v", decoded.CreatedAt, c.CreatedAt)
	}

	if decoded.ID != c.ID {
		t.Errorf("ID: got %v; want %v", decoded.ID, c.ID)
	}
}

func TestCursor_EmptyDecodesToNil(t *testing.T) {
	c, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("decode empty failed : %v", err)
	}

	if c != nil {
		t.Errorf("emtpy cursor should decode to nil, got %v", c)
	}
}

func TestCursor_Malformed(t *testing.T) {
	cases := []string{
		"not-base64-at-all!@#$",
		"YWJjZGVm",             // base64 of "abcdef" — not "ts|id"
		"MjAyNi0wNS0xM1QxNDoz", // base64 partial
	}

	for _, s := range cases {
		if _, err := DecodeCursor(s); err == nil {
			t.Errorf("DecodeCursor(%q) should produce err", s)
		}
	}
}
