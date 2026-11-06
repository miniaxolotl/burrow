package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"burrow/protocol"
	"github.com/redis/go-redis/v9"
)

var attachmentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func validAttachment(value string, empty bool) bool {
	return empty && value == "" || attachmentPattern.MatchString(value)
}

var reservationStatus = redis.NewScript(`
local owner = redis.call('HGET', KEYS[1], 'hash')
if not owner or owner ~= ARGV[1] then return {} end
return {redis.call('HGET', KEYS[1], 'revoked') or '0',redis.call('HGET', KEYS[1], 'persistent') or '0'}
`)

func (s *Server) handleReservation(w http.ResponseWriter, r *http.Request) {
	owner := requestOwner(w, r)
	if owner == "" {
		return
	}
	id := r.PathValue("id")
	if !validTunnelID.MatchString(id) {
		jsonError(w, "Not found", http.StatusNotFound)
		return
	}
	result, err := reservationStatus.Run(r.Context(), s.registry.redis.client, []string{"tunnel-owner:" + id}, owner).StringSlice()
	if err != nil {
		jsonError(w, "Reservation unavailable", http.StatusServiceUnavailable)
		return
	}
	if len(result) != 2 {
		jsonError(w, "Not found", http.StatusNotFound)
		return
	}
	state := "reserved"
	if result[0] == "1" {
		state = "closed"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(protocol.Reservation{TunnelID: id, State: state, Persistent: result[1] == "1"})
}

var reservePersistent = redis.NewScript(claimOwnerScript + `
redis.call('HSET', KEYS[2], 'persistent', '1')
return 1
`)

func (r *RedisClient) ReservePersistent(ctx context.Context, id, owner string) error {
	if owner == "" {
		return ErrTunnelOwnership
	}
	return ownershipResult(reservePersistent.Run(ctx, r.client, []string{"tunnel:" + id, "tunnel-owner:" + id}, owner).Int())
}

// The session registry has already rejected live transports before this commit.
// Non-expiring owner state is authoritative even after metadata TTL or restart.
const checkPersistentScript = `
if redis.call('HGET', KEYS[2], 'persistent') ~= '1' then return -3 end
local attachment = redis.call('HGET', KEYS[2], 'attachment')
if attachment and attachment ~= ARGV[2] then return -4 end
local generation = redis.call('HGET', KEYS[2], 'generation') or ''
local nonce = redis.call('HGET', KEYS[2], 'nonce')
if nonce ~= ARGV[4] then
  if generation ~= ARGV[3] then return -4 end
  generation = ARGV[5]
end
`

var checkPersistent = redis.NewScript(claimOwnerScript + checkPersistentScript + `return 1`)

func (r *RedisClient) CheckPersistent(ctx context.Context, id, owner, attachment, expected, nonce string) error {
	return ownershipResult(checkPersistent.Run(ctx, r.client, []string{"tunnel:" + id, "tunnel-owner:" + id}, owner, attachment, expected, nonce, "").Int())
}

var commitPersistent = redis.NewScript(claimOwnerScript + checkPersistentScript + `
local data = cjson.decode(ARGV[6])
data.session_id = generation
redis.call('HSET', KEYS[2], 'attachment', ARGV[2], 'generation', generation, 'nonce', ARGV[4])
redis.call('SET', KEYS[1], cjson.encode(data), 'PX', ARGV[7])
return generation
`)

func (r *RedisClient) SetPersistent(ctx context.Context, id string, port uint16, owner, attachment, expected, nonce, proposed string) (string, error) {
	if owner == "" || attachment == "" || nonce == "" || proposed == "" {
		return "", ErrStaleAttachment
	}
	ttl := 24 * time.Hour
	data, err := json.Marshal(TunnelData{ID: id, Port: port, OwnerHash: owner, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(ttl)})
	if err != nil {
		return "", err
	}
	result, err := commitPersistent.Run(ctx, r.client, []string{"tunnel:" + id, "tunnel-owner:" + id}, owner, attachment, expected, nonce, proposed, data, ttl.Milliseconds()).Result()
	if err != nil {
		return "", err
	}
	if code, ok := result.(int64); ok {
		return "", ownershipResult(int(code), nil)
	}
	if generation, ok := result.(string); ok && generation != "" {
		return generation, nil
	}
	return "", fmt.Errorf("invalid attachment commit response")
}
