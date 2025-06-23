package auth

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Value     string    `json:"value" db:"value"` // 40-character hexadecimal (base-16); ASSUMING it's a one-time generated token
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

func (t Token) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
