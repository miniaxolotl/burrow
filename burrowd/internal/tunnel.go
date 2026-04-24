package internal

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"burrow/protocol"
)

type TunnelRegistry struct {
	redis    *RedisClient
	hostname string
	domain   string
	mu       sync.RWMutex
	active   map[string]chan protocol.Message
}

func NewTunnelRegistry(redis *RedisClient, hostname, domain string) *TunnelRegistry {
	return &TunnelRegistry{
		redis:    redis,
		hostname: hostname,
		domain:   domain,
		active:   make(map[string]chan protocol.Message),
	}
}

func (t *TunnelRegistry) Register(ctx context.Context, tunnelID string, port uint16) (string, error) {
	ttl := 24 * time.Hour

	if err := t.redis.SetTunnel(ctx, tunnelID, port, ttl); err != nil {
		return "", fmt.Errorf("failed to register tunnel: %w", err)
	}

	t.mu.Lock()
	t.active[tunnelID] = make(chan protocol.Message, 100)
	t.mu.Unlock()

	return fmt.Sprintf("https://%s.%s", tunnelID, t.domain), nil
}

func (t *TunnelRegistry) Get(tunnelID string) (uint16, bool) {
	ctx := context.Background()
	data, err := t.redis.GetTunnel(ctx, tunnelID)
	if err != nil || data == nil {
		return 0, false
	}
	return data.Port, true
}

func (t *TunnelRegistry) Remove(tunnelID string) error {
	ctx := context.Background()

	t.mu.Lock()
	if ch, ok := t.active[tunnelID]; ok {
		close(ch)
		delete(t.active, tunnelID)
	}
	t.mu.Unlock()

	return t.redis.DeleteTunnel(ctx, tunnelID)
}

func (t *TunnelRegistry) List(ctx context.Context) ([]*protocol.TunnelInfo, error) {
	tunnels, err := t.redis.ListTunnels(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*protocol.TunnelInfo, 0, len(tunnels))
	for _, td := range tunnels {
		info := &protocol.TunnelInfo{
			TunnelID: td.ID,
			Port:     td.Port,
			URL:      fmt.Sprintf("https://%s.%s", td.ID, t.domain),
			Status:   "active",
		}
		result = append(result, info)
	}

	return result, nil
}

var tunnelIDAdjectives = []string{
	"arcane", "ancient", "astral", "bold", "brave", "chaotic", "cryptic", "dark", "elder",
	"ethereal", "fierce", "frozen", "hidden", "icy", "jade", "keen", "liquid", "mystic",
	"noble", "obscure", "potent", "quick", "radiant", "shadow", "swift", "twilight",
	"uncanny", "vivid", "wandering", "wild",
}

var tunnelIDNouns = []string{
	"amulet", "basilisk", "cipher", "dragon", "ember", "fortress", "gargoyle", "helm",
	"illusion", "kraken", "lich", "mithril", "nymph", "oracle", "phoenix", "quest",
	"rune", "specter", "talisman", "umbral", "void", "wyrm", "zephyr", "amethyst",
	"bramble", "crypt", "druid", "forge", "grimoire", "haven", "isle", "knave",
	"lava", "moon", "nexus", "obsidian", "prism", "quill", "shadow", "tome", "umbra",
	"vestige", "warden", "xorn", "zinc",
}

func randomTunnelID() string {
	adj := tunnelIDAdjectives[rand.Intn(len(tunnelIDAdjectives))]
	noun := tunnelIDNouns[rand.Intn(len(tunnelIDNouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(100))
}

func (t *TunnelRegistry) GenerateHumanReadableID() string {
	for {
		id := randomTunnelID()

		t.mu.RLock()
		_, exists := t.active[id]
		t.mu.RUnlock()

		if !exists {
			return id
		}
	}
}

func (t *TunnelRegistry) GetChannel(tunnelID string) (chan protocol.Message, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ch, ok := t.active[tunnelID]
	return ch, ok
}

func (t *TunnelRegistry) SendToTunnel(tunnelID string, msg protocol.Message) bool {
	t.mu.RLock()
	ch, ok := t.active[tunnelID]
	t.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case ch <- msg:
		return true
	default:
		return false
	}
}

func (t *TunnelRegistry) HandleWildcardTunnelID(ctx context.Context) (string, error) {
	for attempts := 0; attempts < 100; attempts++ {
		id := randomTunnelID()
		existing, _ := t.redis.GetTunnel(ctx, id)
		if existing == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique tunnel ID after 100 attempts")
}

func ParseTunnelID(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-2], ".")
	}
	return ""
}
