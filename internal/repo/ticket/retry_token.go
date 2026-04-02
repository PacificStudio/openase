package ticket

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func NewRetryToken() string {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		panic(err)
	}
	return hex.EncodeToString(token)
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
