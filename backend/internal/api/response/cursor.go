package response

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cursor is the typed pagination cursor used across list endpoints.
// Encoded as base64("RFC3339Nano|UUID") for opacity.
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// EncodeCursor returns the base64 representation of a cursor.
// Returns "" for the empty cursor (used to mean "no more pages").
func (c *Cursor) EncodeCursor() string {
	if c == nil {
		return ""
	}
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor string. An empty string returns (nil, nil) —
// the caller treats it as "start from the beginning."
func DecodeCursor(s string) (*Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid cursor format")
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor timestamp: %w", err)
	}

	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid cursor id: %w", err)
	}
	return &Cursor{
		CreatedAt: t,
		ID:        id,
	}, nil
}
