package protocol

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TokenMaxAge is how long a generated token remains valid.
const TokenMaxAge = 30 * 24 * time.Hour

// GenerateToken creates a time-stamped HMAC token. Format: "{unix_ts}.{mac}"
func GenerateToken(secret string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return fmt.Sprintf("%s.%s", ts, tokenMAC(secret, ts))
}

// ValidateToken checks the token's HMAC and that it was issued within TokenMaxAge.
func ValidateToken(token, secret string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	if time.Since(time.Unix(ts, 0)) > TokenMaxAge {
		return false
	}
	expected := tokenMAC(secret, parts[0])
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

func tokenMAC(secret, timestamp string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte("burrow-auth:" + timestamp))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}
