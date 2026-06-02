package internal

import (
	"time"

	"github.com/IBM/sarama"
)

type User struct {
	ID               int64      `json:"id"`
	UUID             string     `json:"uuid"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	Role             string     `json:"role"`
	Plan             string     `json:"plan"`
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

type Tunnel struct {
	ID        int64      `json:"id"`
	TunnelID  string     `json:"tunnel_id"`
	Port      int        `json:"port"`
	ClientIP  string     `json:"client_ip"`
	UserID    *int64     `json:"user_id,omitempty"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
}

type TunnelStat struct {
	ID           int64     `json:"id"`
	TunnelID     string    `json:"tunnel_id"`
	BucketStart  time.Time `json:"bucket_start"`
	BytesIn      int64     `json:"bytes_in"`
	BytesOut     int64     `json:"bytes_out"`
	RequestCount int64     `json:"request_count"`
}

type TunnelLog struct {
	ID         int64     `json:"id"`
	TunnelID   string    `json:"tunnel_id"`
	Timestamp  time.Time `json:"timestamp"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	Size       int64     `json:"size"`
	Duration   string    `json:"duration"`
	IP         string    `json:"ip"`
}

type Session struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type Subscription struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"user_id"`
	StripeSubscriptionID string    `json:"stripe_subscription_id"`
	StripePriceID        string    `json:"stripe_price_id"`
	Status               string    `json:"status"`
	CurrentPeriodStart   time.Time `json:"current_period_start"`
	CurrentPeriodEnd     time.Time `json:"current_period_end"`
	CancelAtPeriodEnd    bool      `json:"cancel_at_period_end"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type DashboardStats struct {
	ActiveTunnels int   `json:"active_tunnels"`
	TotalRequests int64 `json:"total_requests"`
	TotalBytesIn  int64 `json:"total_bytes_in"`
	TotalBytesOut int64 `json:"total_bytes_out"`
}

type TunnelLimits struct {
	AnonLimit int `json:"anon_limit"`
	UserLimit int `json:"user_limit"`
	PaidLimit int `json:"paid_limit"`
}

type ServerConfig struct {
	Registry       *TunnelRegistry
	Domain         string
	Secret         string
	Secure         bool
	DeploymentMode string
	PG             *PostgresClient
	JWTSecret      string
	Limits         TunnelLimits
	KafkaProducer  sarama.AsyncProducer
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeProPriceID    string
}