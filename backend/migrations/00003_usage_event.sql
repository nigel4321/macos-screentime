-- +goose Up
-- +goose StatementBegin
CREATE TABLE usage_event (
    id               BIGSERIAL,
    device_id        UUID NOT NULL REFERENCES device(id) ON DELETE CASCADE,
    client_event_id  TEXT NOT NULL,
    bundle_id        TEXT NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL,
    ended_at         TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, started_at),
    UNIQUE (device_id, client_event_id, started_at)
) PARTITION BY RANGE (started_at);

CREATE INDEX idx_usage_event_device_started
    ON usage_event (device_id, started_at DESC);
-- +goose StatementEnd

-- Monthly partitions are created by db.EnsureMonthPartition (see internal/db/partition.go),
-- which application startup invokes for the current and next month.

-- +goose Down
-- +goose StatementBegin
DROP TABLE usage_event;
-- +goose StatementEnd
