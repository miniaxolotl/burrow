CREATE TABLE tunnels (
    id BIGSERIAL PRIMARY KEY,
    tunnel_id TEXT NOT NULL UNIQUE,
    port INTEGER NOT NULL,
    client_ip TEXT NOT NULL DEFAULT '',
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'stopped')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ
);

CREATE INDEX idx_tunnels_client_ip_status ON tunnels (client_ip, status);
CREATE INDEX idx_tunnels_user_id_status ON tunnels (user_id) WHERE status = 'active';