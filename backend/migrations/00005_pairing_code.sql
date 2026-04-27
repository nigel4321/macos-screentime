-- +goose Up
-- +goose StatementBegin
CREATE TABLE pairing_code (
    code         TEXT PRIMARY KEY CHECK (code ~ '^[0-9]{6}$'),
    account_id   UUID NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ
);

CREATE INDEX idx_pairing_code_active
    ON pairing_code (expires_at)
    WHERE consumed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE pairing_code;
-- +goose StatementEnd
