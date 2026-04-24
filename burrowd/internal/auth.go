package internal

import "burrow/protocol"

func GenerateToken(secret string) string      { return protocol.GenerateToken(secret) }
func ValidateToken(token, secret string) bool { return protocol.ValidateToken(token, secret) }
