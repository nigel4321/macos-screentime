-- +goose Up
-- +goose StatementBegin
CREATE TABLE device (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id         UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    platform           TEXT NOT NULL CHECK (platform IN ('macos', 'android')),
    fingerprint        TEXT NOT NULL,
    device_token_hash  BYTEA NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at       TIMESTAMPTZ,
    UNIQUE (account_id, fingerprint)
);

CREATE INDEX idx_device_account ON device(account_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE device;
-- +goose StatementEnd
