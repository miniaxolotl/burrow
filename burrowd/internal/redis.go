package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

type TunnelData struct {
	ID        string    `json:"id"`
	Port      uint16    `json:"port"`
	OwnerHash string    `json:"owner_hash,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewRedisClient(addr string) (*RedisClient, error) {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		opt = &redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		}
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

var (
	ErrTunnelOwnership = errors.New("tunnel ID is unavailable")
	ErrTunnelRevoked   = errors.New("tunnel has been closed")
)

// Every ownership operation uses the same migration rule. Reservations have no
// TTL: a shared URL must not become claimable when its connection record expires.
const reserveOwnerScript = `
local function reserveOwner()
  local owner = redis.call('HGET', KEYS[2], 'hash')
  if owner then return owner end
  local previous = redis.call('GET', KEYS[1])
  if not previous then return nil end
  owner = cjson.decode(previous).owner_hash or ''
  redis.call('HSET', KEYS[2], 'hash', owner)
  if owner == '' then redis.call('HSET', KEYS[2], 'revoked', '1') end
  return owner
end
`

const claimOwnerScript = reserveOwnerScript + `
local owner = reserveOwner()
if owner then
  if owner ~= ARGV[1] then return -1 end
  if redis.call('HGET', KEYS[2], 'revoked') == '1' then return -2 end
else
  redis.call('HSET', KEYS[2], 'hash', ARGV[1])
end
`

var reserveTunnel = redis.NewScript(claimOwnerScript + `return 1`)

// ReserveTunnel claims only the durable owner, leaving any live generation and
// its keepalive intact until the replacement transport is ready.
func (r *RedisClient) ReserveTunnel(ctx context.Context, id, ownerHash string) error {
	if ownerHash == "" {
		return ErrTunnelOwnership
	}
	return ownershipResult(reserveTunnel.Run(ctx, r.client, []string{"tunnel:" + id, "tunnel-owner:" + id}, ownerHash).Int())
}

// Installing a generation repeats the ownership/revocation check atomically.
var claimTunnel = redis.NewScript(claimOwnerScript + `
redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3])
return 1
`)

func (r *RedisClient) SetTunnel(ctx context.Context, id string, port uint16, ownerHash, sessionID string, ttl time.Duration) error {
	if ownerHash == "" || sessionID == "" {
		return ErrTunnelOwnership
	}
	key := fmt.Sprintf("tunnel:%s", id)
	data := TunnelData{
		ID:        id,
		Port:      port,
		OwnerHash: ownerHash,
		SessionID: sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel data: %w", err)
	}
	return ownershipResult(claimTunnel.Run(ctx, r.client, []string{key, "tunnel-owner:" + id}, ownerHash, b, ttl.Milliseconds()).Int())
}

func ownershipResult(result int, err error) error {
	if err != nil {
		return err
	}
	switch result {
	case -1:
		return ErrTunnelOwnership
	case -2:
		return ErrTunnelRevoked
	case 1:
		return nil
	default:
		return fmt.Errorf("unexpected ownership result: %d", result)
	}
}

func (r *RedisClient) GetTunnel(ctx context.Context, id string) (*TunnelData, error) {
	key := fmt.Sprintf("tunnel:%s", id)
	b, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var data TunnelData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tunnel data: %w", err)
	}
	return &data, nil
}

var releaseTunnel = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if not value or cjson.decode(value).session_id ~= ARGV[1] then return 0 end
return redis.call('DEL', KEYS[1])
`)

func (r *RedisClient) DeleteTunnel(ctx context.Context, id, sessionID string) error {
	return releaseTunnel.Run(ctx, r.client, []string{"tunnel:" + id}, sessionID).Err()
}

var refreshTunnel = redis.NewScript(`
local value = redis.call('GET', KEYS[1])
if not value or cjson.decode(value).session_id ~= ARGV[1] then return 0 end
local data = cjson.decode(value)
data.expires_at = ARGV[3]
redis.call('SET', KEYS[1], cjson.encode(data), 'PX', ARGV[2])
return 1
`)

func (r *RedisClient) RefreshTunnel(ctx context.Context, id, sessionID string, ttl time.Duration) error {
	result, err := refreshTunnel.Run(ctx, r.client, []string{"tunnel:" + id}, sessionID, ttl.Milliseconds(), time.Now().Add(ttl).Format(time.RFC3339Nano)).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return ErrTunnelRevoked
	}
	return nil
}

var revokeTunnel = redis.NewScript(reserveOwnerScript + `
local owner = reserveOwner()
if not owner or owner == '' or owner ~= ARGV[1] then return 0 end
redis.call('HSET', KEYS[2], 'hash', owner, 'revoked', '1')
redis.call('DEL', KEYS[1])
return 1
`)

func (r *RedisClient) RevokeTunnel(ctx context.Context, id, ownerHash string) error {
	result, err := revokeTunnel.Run(ctx, r.client, []string{"tunnel:" + id, "tunnel-owner:" + id}, ownerHash).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return ErrTunnelOwnership
	}
	return nil
}

var reserveLegacyOwner = redis.NewScript(reserveOwnerScript + `
reserveOwner()
return 1
`)

// ReserveLegacyOwners runs before serving requests so pre-upgrade records cannot
// expire into claimable public URLs. Tokenless legacy IDs are retired permanently.
func (r *RedisClient) ReserveLegacyOwners(ctx context.Context) error {
	var cursor uint64
	for {
		batch, next, err := r.client.Scan(ctx, cursor, "tunnel:*", 100).Result()
		if err != nil {
			return err
		}
		for _, key := range batch {
			id := strings.TrimPrefix(key, "tunnel:")
			if err := reserveLegacyOwner.Run(ctx, r.client, []string{key, "tunnel-owner:" + id}).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
