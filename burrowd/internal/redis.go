package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

type TunnelData struct {
	ID        string    `json:"id"`
	Port      uint16    `json:"port"`
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

func (r *RedisClient) SetTunnel(ctx context.Context, id string, port uint16, ttl time.Duration) error {
	key := fmt.Sprintf("tunnel:%s", id)
	data := TunnelData{
		ID:        id,
		Port:      port,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal tunnel data: %w", err)
	}
	return r.client.Set(ctx, key, b, ttl).Err()
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

func (r *RedisClient) DeleteTunnel(ctx context.Context, id string) error {
	key := fmt.Sprintf("tunnel:%s", id)
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) ListTunnels(ctx context.Context) ([]*TunnelData, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := r.client.Scan(ctx, cursor, "tunnel:*", 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	tunnels := make([]*TunnelData, 0, len(keys))
	for _, key := range keys {
		id := key[7:]
		data, err := r.GetTunnel(ctx, id)
		if err != nil {
			continue
		}
		if data != nil {
			tunnels = append(tunnels, data)
		}
	}
	return tunnels, nil
}
