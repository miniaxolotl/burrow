CREATE TABLE tunnel_stats (
    id BIGSERIAL PRIMARY KEY,
    tunnel_id TEXT NOT NULL REFERENCES tunnels(tunnel_id),
    bucket_start TIMESTAMPTZ NOT NULL,
    bytes_in BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    UNIQUE (tunnel_id, bucket_start)
);

CREATE INDEX idx_tunnel_stats_tunnel_id ON tunnel_stats (tunnel_id, bucket_start DESC);